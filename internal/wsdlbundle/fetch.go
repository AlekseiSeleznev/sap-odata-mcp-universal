package wsdlbundle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	wsdl11Namespace   = "http://schemas.xmlsoap.org/wsdl/"
	wsdl20Namespace   = "http://www.w3.org/ns/wsdl"
	xsdNamespace      = "http://www.w3.org/2001/XMLSchema"
	soap11Namespace   = "http://schemas.xmlsoap.org/wsdl/soap/"
	soap12Namespace   = "http://schemas.xmlsoap.org/wsdl/soap12/"
	wsp15Namespace    = "http://www.w3.org/ns/ws-policy"
	wsp12Namespace    = "http://schemas.xmlsoap.org/ws/2004/09/policy"
	xincludeNamespace = "http://www.w3.org/2001/XInclude"
)

type stopError struct {
	Phase   string
	Code    string
	Message string
}

type fetchResult struct {
	Bundle             *Bundle
	NetworkGetsStarted int
	TLSPeerSHA256      *string
}

type queueItem struct {
	RequestURI            string
	NormalizedKey         string
	Depth                 int
	InheritedXSDNamespace string
}

type resolvedURI struct {
	RequestURI    string
	NormalizedKey string
}

type rawDocument struct {
	URI       string
	ID        string
	Kind      string
	MediaType string
	Raw       []byte
	Parsed    parsedDocument
}

type reference struct {
	Raw                   string
	Relation              string
	InheritedXSDNamespace string
}

type parsedDocument struct {
	Kind             string
	TargetNamespace  string
	References       []reference
	IDs              map[string]bool
	PortTypes        map[string]map[string]operationInfo
	Bindings         map[string]bindingInfo
	Services         map[string][]portInfo
	Messages         map[string][]MessagePart
	XSDComponents    []XSDComponent
	PolicyAssertions []string
}

type operationInfo struct {
	Input  string
	Output string
	Faults []string
}

type bindingInfo struct {
	TypeQName   string
	SOAPVersion string
	Actions     map[string]string
}

type portInfo struct {
	Name         string
	BindingQName string
	SOAPVersion  string
}

func fetchClosure(ctx context.Context, manifest Manifest, credentials Credentials, evidenceKey []byte, roundTripper http.RoundTripper) (fetchResult, *stopError) {
	result := fetchResult{}
	root, err := resolveFetchURI(manifest.RootURL, nil, manifest)
	if err != nil {
		return result, stop("uri", "URI_POLICY_VIOLATION", "sealed root violates URI policy")
	}
	queue := []queueItem{{RequestURI: root.RequestURI, NormalizedKey: root.NormalizedKey, Depth: 0}}
	seen := map[string]bool{}
	inheritedByURI := map[string]string{root.NormalizedKey: ""}
	documents := map[string]*rawDocument{}
	requiredFragments := map[string]map[string]bool{}
	edges := make([]Edge, 0)
	totalBytes := int64(0)
	referenceCount := 0

	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return result, stop("timeout", "WHOLE_ACTION_TIMEOUT", "whole-action timeout reached")
		}
		item := queue[0]
		queue = queue[1:]
		if seen[item.NormalizedKey] {
			continue
		}
		if item.Depth > manifest.Limits.MaxDepth {
			return result, stop("closure", "DEPTH_LIMIT", "recursive closure depth limit exceeded")
		}
		if len(seen) >= manifest.Limits.MaxDocuments {
			return result, stop("closure", "DOCUMENT_LIMIT", "document count limit exceeded")
		}
		seen[item.NormalizedKey] = true
		result.NetworkGetsStarted++
		doc, tlsPeer, fetchStop := fetchDocument(ctx, item.RequestURI, manifest, credentials, roundTripper)
		if fetchStop != nil {
			return result, fetchStop
		}
		if tlsPeer != nil {
			if result.TLSPeerSHA256 != nil && *result.TLSPeerSHA256 != *tlsPeer {
				return result, stop("identity", "TLS_IDENTITY_CHANGED", "TLS peer identity changed inside one action")
			}
			result.TLSPeerSHA256 = tlsPeer
		}
		if item.InheritedXSDNamespace != "" && doc.Kind == "XSD10" {
			if doc.Parsed.TargetNamespace != "" && doc.Parsed.TargetNamespace != item.InheritedXSDNamespace {
				return result, stop("contract", "CONTRACT_CONFLICT", "included XSD target namespace conflicts with its parent schema")
			}
			doc.Parsed.TargetNamespace = item.InheritedXSDNamespace
			for index := range doc.Parsed.XSDComponents {
				if doc.Parsed.XSDComponents[index].Namespace == "" {
					doc.Parsed.XSDComponents[index].Namespace = item.InheritedXSDNamespace
					if strings.HasPrefix(doc.Parsed.XSDComponents[index].ParentQName, "{}") {
						doc.Parsed.XSDComponents[index].ParentQName = qname(item.InheritedXSDNamespace, strings.TrimPrefix(doc.Parsed.XSDComponents[index].ParentQName, "{}"))
					}
				}
			}
			for index := range doc.Parsed.References {
				if (doc.Parsed.References[index].Relation == "xsd_include" || doc.Parsed.References[index].Relation == "xsd_redefine") && doc.Parsed.References[index].InheritedXSDNamespace == "" {
					doc.Parsed.References[index].InheritedXSDNamespace = item.InheritedXSDNamespace
				}
			}
		}
		totalBytes += int64(len(doc.Raw))
		if totalBytes > manifest.Limits.MaxTotalBytes {
			return result, stop("limits", "TOTAL_SIZE_LIMIT", "total raw byte limit exceeded")
		}
		doc.ID = documentID(evidenceKey, item.NormalizedKey)
		documents[item.NormalizedKey] = doc
		if fragments := requiredFragments[item.NormalizedKey]; len(fragments) > 0 {
			for fragment := range fragments {
				if !doc.Parsed.IDs[fragment] {
					return result, stop("xml", "POLICY_FRAGMENT_UNRESOLVED", "policy fragment did not resolve")
				}
			}
		}

		for _, ref := range doc.Parsed.References {
			referenceCount++
			if referenceCount > manifest.Limits.MaxReferences {
				return result, stop("closure", "REFERENCE_LIMIT", "reference count limit exceeded")
			}
			target, fragment, sameDocument, err := resolveReference(item.RequestURI, ref.Raw, manifest)
			if err != nil {
				return result, stop("uri", "URI_POLICY_VIOLATION", "document reference violates URI policy")
			}
			if sameDocument {
				if fragment == "" || !doc.Parsed.IDs[fragment] {
					return result, stop("xml", "POLICY_FRAGMENT_UNRESOLVED", "same-document policy fragment did not resolve")
				}
				continue
			}
			if fragment != "" {
				if requiredFragments[target.NormalizedKey] == nil {
					requiredFragments[target.NormalizedKey] = map[string]bool{}
				}
				requiredFragments[target.NormalizedKey][fragment] = true
			}
			targetID := documentID(evidenceKey, target.NormalizedKey)
			edges = append(edges, Edge{FromDocumentID: doc.ID, ToDocumentID: targetID, Relation: ref.Relation})
			loadedTarget := documents[target.NormalizedKey]
			if loadedTarget != nil && loadedTarget.Parsed.TargetNamespace != "" {
				if ref.InheritedXSDNamespace != "" && loadedTarget.Parsed.TargetNamespace != ref.InheritedXSDNamespace {
					return result, stop("contract", "CONTRACT_CONFLICT", "included XSD target namespace conflicts with its parent schema")
				}
			} else if inherited, known := inheritedByURI[target.NormalizedKey]; known && inherited != ref.InheritedXSDNamespace {
				return result, stop("contract", "CONTRACT_CONFLICT", "one XSD document was referenced with conflicting inherited namespaces")
			} else if !known {
				inheritedByURI[target.NormalizedKey] = ref.InheritedXSDNamespace
			}
			if !seen[target.NormalizedKey] {
				queue = append(queue, queueItem{RequestURI: target.RequestURI, NormalizedKey: target.NormalizedKey, Depth: item.Depth + 1, InheritedXSDNamespace: ref.InheritedXSDNamespace})
			}
		}
	}

	for uri, fragments := range requiredFragments {
		doc := documents[uri]
		if doc == nil {
			return result, stop("closure", "CLOSURE_INCOMPLETE", "referenced document is missing")
		}
		for fragment := range fragments {
			if !doc.Parsed.IDs[fragment] {
				return result, stop("xml", "POLICY_FRAGMENT_UNRESOLVED", "policy fragment did not resolve")
			}
		}
	}

	contract, contractStop := buildContract(ctx, manifest, documents, evidenceKey)
	if contractStop != nil {
		return result, contractStop
	}
	docEvidence := make([]DocumentEvidence, 0, len(documents))
	for _, doc := range documents {
		if ctx.Err() != nil {
			return result, stop("timeout", "WHOLE_ACTION_TIMEOUT", "whole-action timeout reached")
		}
		rawDigest := sha256.Sum256(doc.Raw)
		sanitized := sanitizedDocumentSummary(doc.ID, doc.Parsed, manifest, evidenceKey)
		sanitizedBytes, err := json.Marshal(sanitized)
		if err != nil {
			return result, stop("sanitize", "SANITIZATION_FAILED", "sanitized document summary failed")
		}
		sanitizedDigest := sha256.Sum256(sanitizedBytes)
		docEvidence = append(docEvidence, DocumentEvidence{
			DocumentID: doc.ID, Kind: doc.Kind, ByteCount: len(doc.Raw),
			RawSHA256: hex.EncodeToString(rawDigest[:]), SanitizedSHA256: hex.EncodeToString(sanitizedDigest[:]), MediaType: doc.MediaType,
		})
	}
	sort.Slice(docEvidence, func(i, j int) bool { return docEvidence[i].DocumentID < docEvidence[j].DocumentID })
	edges = canonicalEdges(edges)
	bundle := &Bundle{Complete: true, RootDocumentID: documentID(evidenceKey, root.NormalizedKey), Documents: docEvidence, Edges: edges, Contract: *contract}
	bundleSHA, err := bundleSHA256(bundle.RootDocumentID, bundle.Documents, bundle.Edges)
	if err != nil {
		return result, stop("digest", "DIGEST_FAILED", "bundle digest failed")
	}
	bundle.BundleSHA256 = bundleSHA
	if ctx.Err() != nil {
		return result, stop("timeout", "WHOLE_ACTION_TIMEOUT", "whole-action timeout reached")
	}
	if leak := bundleLeak(manifest, credentials, bundle); leak {
		return result, stop("sanitize", "SANITIZATION_FAILED", "forbidden private value remained in evidence")
	}
	result.Bundle = bundle
	return result, nil
}

func bundleSHA256(rootDocumentID string, documents []DocumentEvidence, edges []Edge) (string, error) {
	projection := struct {
		RootDocumentID string             `json:"root_document_id"`
		Documents      []DocumentEvidence `json:"documents"`
		Edges          []Edge             `json:"edges"`
	}{rootDocumentID, documents, edges}
	canonical, err := canonicalJSON(projection)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func fetchDocument(ctx context.Context, rawURI string, manifest Manifest, credentials Credentials, roundTripper http.RoundTripper) (*rawDocument, *string, *stopError) {
	docCtx, cancel := context.WithTimeout(ctx, time.Duration(manifest.Limits.PerDocumentTimeoutMS)*time.Millisecond)
	defer cancel()
	request, err := http.NewRequestWithContext(docCtx, http.MethodGet, rawURI, nil)
	if err != nil {
		return nil, nil, stop("request", "REQUEST_INVALID", "sealed request could not be built")
	}
	request.Header.Set("Accept", "application/wsdl+xml, application/xsd+xml, application/xml, text/xml")
	request.Header.Set("Accept-Encoding", "identity")
	request.SetBasicAuth(credentials.Username, credentials.Password)
	response, err := roundTripper.RoundTrip(request)
	if err != nil {
		if docCtx.Err() != nil {
			return nil, nil, stop("network", "DOCUMENT_TIMEOUT", "document request timed out")
		}
		return nil, nil, stop("network", "NETWORK_ERROR", "document transport failed")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, nil, stop("auth", "AUTH_FAILURE", "SAP authentication or authorization failed")
	}
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		return nil, nil, stop("http", "HTTP_REDIRECT", "redirects are forbidden")
	}
	if response.StatusCode != http.StatusOK {
		return nil, nil, stop("http", "HTTP_STATUS", "non-success HTTP status")
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	mediaType = strings.ToLower(mediaType)
	if err != nil || !isXMLMediaType(mediaType) {
		return nil, nil, stop("http", "MEDIA_TYPE_INVALID", "response is not an allowed XML media type")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, manifest.Limits.MaxDocumentBytes+1))
	if err != nil {
		return nil, nil, stop("network", "BODY_READ_FAILED", "response body could not be read")
	}
	if int64(len(body)) > manifest.Limits.MaxDocumentBytes {
		return nil, nil, stop("limits", "DOCUMENT_SIZE_LIMIT", "document byte limit exceeded")
	}
	if len(body) == 0 || !utf8.Valid(body) {
		return nil, nil, stop("xml", "XML_INVALID", "document is empty or not UTF-8")
	}
	parsed, parseStop := parseXMLDocument(docCtx, body, manifest.Limits)
	if parseStop != nil {
		if ctx.Err() != nil {
			return nil, nil, stop("timeout", "WHOLE_ACTION_TIMEOUT", "whole-action timeout reached")
		}
		return nil, nil, parseStop
	}
	var tlsPeer *string
	if response.TLS != nil && len(response.TLS.PeerCertificates) > 0 {
		digest := sha256.Sum256(response.TLS.PeerCertificates[0].Raw)
		value := hex.EncodeToString(digest[:])
		tlsPeer = &value
	}
	return &rawDocument{URI: rawURI, Kind: parsed.Kind, MediaType: mediaType, Raw: body, Parsed: parsed}, tlsPeer, nil
}

func isXMLMediaType(mediaType string) bool {
	return mediaType == "application/xml" || mediaType == "text/xml" || mediaType == "application/wsdl+xml" || mediaType == "application/xsd+xml" || strings.HasSuffix(mediaType, "+xml")
}

func parseXMLDocument(ctx context.Context, body []byte, limits Limits) (parsedDocument, *stopError) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.Strict = true
	parsed := parsedDocument{IDs: map[string]bool{}, PortTypes: map[string]map[string]operationInfo{}, Bindings: map[string]bindingInfo{}, Services: map[string][]portInfo{}, Messages: map[string][]MessagePart{}}
	namespaces := []map[string]string{{}}
	depth := 0
	tokens := 0
	var rootSeen bool
	var currentPortType, currentBinding, currentService string
	var currentPortTypeDepth, currentBindingDepth, currentServiceDepth int
	var currentPTOperation, currentBindingOperation string
	var currentPTOperationDepth, currentBindingOperationDepth int
	var currentMessage string
	var currentMessageDepth int
	componentAtDepth := map[int]int{}
	childOrder := map[int]int{}
	schemaTargetAtDepth := map[int]string{}
	policyDepth := 0
	redefineDepth := 0

	for {
		if ctx.Err() != nil {
			return parsed, stop("timeout", "DOCUMENT_TIMEOUT", "document processing timed out")
		}
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return parsed, stop("xml", "XML_INVALID", "XML parsing failed")
		}
		tokens++
		if tokens > limits.MaxXMLTokens {
			return parsed, stop("limits", "XML_TOKEN_LIMIT", "XML token limit exceeded")
		}
		switch value := token.(type) {
		case xml.Directive:
			upper := strings.ToUpper(string(value))
			if strings.Contains(upper, "DOCTYPE") || strings.Contains(upper, "ENTITY") {
				return parsed, stop("xml", "XML_UNSAFE", "DOCTYPE and entity declarations are forbidden")
			}
		case xml.StartElement:
			depth++
			if depth > limits.MaxXMLNesting {
				return parsed, stop("limits", "XML_NESTING_LIMIT", "XML nesting limit exceeded")
			}
			if len(value.Attr) > limits.MaxAttributes {
				return parsed, stop("limits", "XML_ATTRIBUTE_LIMIT", "XML attribute count limit exceeded")
			}
			for _, attr := range value.Attr {
				if len(attr.Value) > limits.MaxAttributeBytes {
					return parsed, stop("limits", "XML_ATTRIBUTE_SIZE_LIMIT", "XML attribute size limit exceeded")
				}
			}
			ns := copyNamespaces(namespaces[len(namespaces)-1])
			for _, attr := range value.Attr {
				if attr.Name.Space == "xmlns" {
					ns[attr.Name.Local] = attr.Value
				} else if attr.Name.Space == "" && attr.Name.Local == "xmlns" {
					ns[""] = attr.Value
				}
			}
			namespaces = append(namespaces, ns)
			if value.Name.Space == wsdl20Namespace || (value.Name.Space == xsdNamespace && isXSD11OnlyElement(value.Name.Local)) ||
				((value.Name.Local == "Policy" || value.Name.Local == "PolicyReference") && strings.Contains(strings.ToLower(value.Name.Space), "policy") && !isPolicyNamespace(value.Name.Space)) {
				return parsed, stop("xml", "UNSUPPORTED_DIALECT", "unsupported WSDL or XSD dialect")
			}
			if value.Name.Space == xincludeNamespace {
				return parsed, stop("xml", "XML_UNSAFE", "XInclude is forbidden")
			}
			if !rootSeen {
				rootSeen = true
				switch {
				case value.Name.Space == wsdl11Namespace && value.Name.Local == "definitions":
					parsed.Kind = "WSDL11"
				case value.Name.Space == xsdNamespace && value.Name.Local == "schema":
					parsed.Kind = "XSD10"
				case isPolicyNamespace(value.Name.Space) && value.Name.Local == "Policy":
					parsed.Kind = "WS_POLICY"
				default:
					return parsed, stop("xml", "UNSUPPORTED_DIALECT", "root XML dialect is unsupported")
				}
			}
			for _, attr := range value.Attr {
				if attr.Name.Local == "Id" || attr.Name.Local == "id" {
					parsed.IDs[attr.Value] = true
				}
			}
			if value.Name.Space == wsdl11Namespace && value.Name.Local == "definitions" {
				parsed.TargetNamespace = attribute(value, "targetNamespace")
			}
			if value.Name.Space == xsdNamespace && value.Name.Local == "schema" {
				target := attribute(value, "targetNamespace")
				schemaTargetAtDepth[depth] = target
				if parsed.Kind == "XSD10" && depth == 1 {
					parsed.TargetNamespace = target
				}
			}
			if ref, relation, required := referenceAttribute(value); relation != "" {
				if required && strings.TrimSpace(ref) == "" {
					return parsed, stop("xml", "REFERENCE_MISSING", "required XML reference is missing")
				}
				if strings.TrimSpace(ref) != "" {
					inheritedNamespace := ""
					if relation == "xsd_include" || relation == "xsd_redefine" {
						inheritedNamespace, _ = nearestString(schemaTargetAtDepth, depth)
					}
					parsed.References = append(parsed.References, reference{Raw: ref, Relation: relation, InheritedXSDNamespace: inheritedNamespace})
				}
			}
			if policyURIs := attribute(value, "PolicyURIs"); policyURIs != "" {
				for _, item := range strings.Fields(policyURIs) {
					parsed.References = append(parsed.References, reference{Raw: item, Relation: "policy_reference"})
				}
			}
			if isPolicyNamespace(value.Name.Space) && value.Name.Local == "Policy" {
				policyDepth = depth
			} else if policyDepth > 0 && value.Name.Space != "" && !isPolicyNamespace(value.Name.Space) && value.Name.Space != "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" {
				parsed.PolicyAssertions = append(parsed.PolicyAssertions, expandedName(value.Name))
			}

			if value.Name.Space == wsdl11Namespace {
				switch value.Name.Local {
				case "message":
					currentMessage = qname(parsed.TargetNamespace, attribute(value, "name"))
					if _, exists := parsed.Messages[currentMessage]; exists {
						return parsed, stop("contract", "CONTRACT_CONFLICT", "duplicate WSDL message definition")
					}
					currentMessageDepth = depth
					parsed.Messages[currentMessage] = []MessagePart{}
				case "part":
					if currentMessage != "" {
						parsed.Messages[currentMessage] = append(parsed.Messages[currentMessage], MessagePart{
							Name: attribute(value, "name"), ElementQName: expandQName(attribute(value, "element"), ns), TypeQName: expandQName(attribute(value, "type"), ns),
						})
					}
				case "portType":
					currentPortType = qname(parsed.TargetNamespace, attribute(value, "name"))
					if _, exists := parsed.PortTypes[currentPortType]; exists {
						return parsed, stop("contract", "CONTRACT_CONFLICT", "duplicate WSDL portType definition")
					}
					currentPortTypeDepth = depth
					parsed.PortTypes[currentPortType] = map[string]operationInfo{}
				case "binding":
					currentBinding = qname(parsed.TargetNamespace, attribute(value, "name"))
					if _, exists := parsed.Bindings[currentBinding]; exists {
						return parsed, stop("contract", "CONTRACT_CONFLICT", "duplicate WSDL binding definition")
					}
					currentBindingDepth = depth
					parsed.Bindings[currentBinding] = bindingInfo{TypeQName: expandQName(attribute(value, "type"), ns), Actions: map[string]string{}}
				case "service":
					currentService = qname(parsed.TargetNamespace, attribute(value, "name"))
					currentServiceDepth = depth
				case "operation":
					if currentPortType != "" {
						currentPTOperation = attribute(value, "name")
						currentPTOperationDepth = depth
						parsed.PortTypes[currentPortType][currentPTOperation] = operationInfo{}
					} else if currentBinding != "" {
						currentBindingOperation = attribute(value, "name")
						currentBindingOperationDepth = depth
					}
				case "input", "output", "fault":
					if currentPortType != "" && currentPTOperation != "" {
						op := parsed.PortTypes[currentPortType][currentPTOperation]
						message := expandQName(attribute(value, "message"), ns)
						if value.Name.Local == "input" {
							op.Input = message
						}
						if value.Name.Local == "output" {
							op.Output = message
						}
						if value.Name.Local == "fault" {
							op.Faults = append(op.Faults, message)
						}
						parsed.PortTypes[currentPortType][currentPTOperation] = op
					}
				case "port":
					if currentService != "" {
						parsed.Services[currentService] = append(parsed.Services[currentService], portInfo{Name: qname(parsed.TargetNamespace, attribute(value, "name")), BindingQName: expandQName(attribute(value, "binding"), ns)})
					}
				}
			}
			if value.Name.Space == soap11Namespace || value.Name.Space == soap12Namespace {
				soapVersion := "1.1"
				if value.Name.Space == soap12Namespace {
					soapVersion = "1.2"
				}
				if value.Name.Local == "binding" && currentBinding != "" {
					binding := parsed.Bindings[currentBinding]
					binding.SOAPVersion = soapVersion
					parsed.Bindings[currentBinding] = binding
				}
				if value.Name.Local == "operation" && currentBinding != "" && currentBindingOperation != "" {
					binding := parsed.Bindings[currentBinding]
					binding.Actions[currentBindingOperation] = attribute(value, "soapAction")
					parsed.Bindings[currentBinding] = binding
				}
				if value.Name.Local == "address" && currentService != "" && len(parsed.Services[currentService]) > 0 {
					ports := parsed.Services[currentService]
					ports[len(ports)-1].SOAPVersion = soapVersion
					parsed.Services[currentService] = ports
				}
			}
			if value.Name.Space == xsdNamespace {
				if value.Name.Local == "redefine" {
					redefineDepth = depth
				}
				if value.Name.Local == "element" || value.Name.Local == "complexType" || value.Name.Local == "simpleType" {
					if name := attribute(value, "name"); name != "" {
						parentQName := ""
						order := 0
						if parentIndex, ok := nearestComponent(componentAtDepth, depth); ok {
							parent := parsed.XSDComponents[parentIndex]
							parentQName = qname(parent.Namespace, parent.Name)
							childOrder[parentIndex]++
							order = childOrder[parentIndex]
						}
						schemaTarget, _ := nearestString(schemaTargetAtDepth, depth)
						component := XSDComponent{Namespace: schemaTarget, ParentQName: parentQName, Name: name, Kind: value.Name.Local, Order: order, Type: expandQName(attribute(value, "type"), ns), MinOccurs: attributeOr(value, "minOccurs", "1"), MaxOccurs: attributeOr(value, "maxOccurs", "1"), Facets: []XSDFacet{}, Redefines: redefineDepth > 0 && depth > redefineDepth}
						parsed.XSDComponents = append(parsed.XSDComponents, component)
						componentAtDepth[depth] = len(parsed.XSDComponents) - 1
					}
				}
				if value.Name.Local == "restriction" {
					if index, ok := nearestComponent(componentAtDepth, depth); ok {
						component := parsed.XSDComponents[index]
						component.Type = expandQName(attribute(value, "base"), ns)
						parsed.XSDComponents[index] = component
					}
				}
				if isFacet(value.Name.Local) {
					if index, ok := nearestComponent(componentAtDepth, depth); ok {
						component := parsed.XSDComponents[index]
						component.Facets = append(component.Facets, XSDFacet{Name: value.Name.Local, Value: attribute(value, "value")})
						parsed.XSDComponents[index] = component
					}
				}
			}
		case xml.EndElement:
			if depth == currentMessageDepth {
				currentMessage = ""
				currentMessageDepth = 0
			}
			if depth == currentPTOperationDepth {
				currentPTOperation = ""
				currentPTOperationDepth = 0
			}
			if depth == currentBindingOperationDepth {
				currentBindingOperation = ""
				currentBindingOperationDepth = 0
			}
			if depth == currentPortTypeDepth {
				currentPortType = ""
				currentPortTypeDepth = 0
			}
			if depth == currentBindingDepth {
				currentBinding = ""
				currentBindingDepth = 0
			}
			if depth == currentServiceDepth {
				currentService = ""
				currentServiceDepth = 0
			}
			if depth == policyDepth {
				policyDepth = 0
			}
			if depth == redefineDepth {
				redefineDepth = 0
			}
			delete(componentAtDepth, depth)
			delete(schemaTargetAtDepth, depth)
			if len(namespaces) > 1 {
				namespaces = namespaces[:len(namespaces)-1]
			}
			depth--
		}
	}
	if !rootSeen || depth != 0 {
		return parsed, stop("xml", "XML_INVALID", "XML document is incomplete")
	}
	parsed.PolicyAssertions = uniqueSorted(parsed.PolicyAssertions)
	return parsed, nil
}

func referenceAttribute(value xml.StartElement) (string, string, bool) {
	if value.Name.Space == wsdl11Namespace && value.Name.Local == "import" {
		return attribute(value, "location"), "wsdl_import", true
	}
	if value.Name.Space == xsdNamespace {
		switch value.Name.Local {
		case "import":
			return attribute(value, "schemaLocation"), "xsd_import", true
		case "include":
			return attribute(value, "schemaLocation"), "xsd_include", true
		case "redefine":
			return attribute(value, "schemaLocation"), "xsd_redefine", true
		}
	}
	if isPolicyNamespace(value.Name.Space) && value.Name.Local == "PolicyReference" {
		return attribute(value, "URI"), "policy_reference", true
	}
	return "", "", false
}

func buildContract(ctx context.Context, manifest Manifest, documents map[string]*rawDocument, key []byte) (*ContractSummary, *stopError) {
	services := map[string][]portInfo{}
	bindings := map[string]bindingInfo{}
	portTypes := map[string]map[string]operationInfo{}
	messages := map[string][]MessagePart{}
	componentIndex := map[string]XSDComponent{}
	originalComponents := map[string]bool{}
	redefinedComponents := map[string]bool{}
	targets := []string{}
	assertions := []string{}
	components := []XSDComponent{}
	xsdElements := map[string]bool{}
	xsdTypes := map[string]bool{}
	for _, uri := range sortedDocumentURIs(documents) {
		if ctx.Err() != nil {
			return nil, stop("timeout", "WHOLE_ACTION_TIMEOUT", "whole-action timeout reached")
		}
		parsed := documents[uri].Parsed
		if parsed.TargetNamespace != "" {
			targets = append(targets, sanitizeValue(parsed.TargetNamespace, manifest, key))
		}
		assertions = append(assertions, sanitizeStrings(parsed.PolicyAssertions, manifest, key)...)
		for name, value := range parsed.Services {
			if existing, ok := services[name]; ok && !reflect.DeepEqual(existing, value) {
				return nil, stop("contract", "CONTRACT_CONFLICT", "WSDL service definitions conflict")
			}
			services[name] = value
		}
		for name, value := range parsed.Bindings {
			if existing, ok := bindings[name]; ok && !reflect.DeepEqual(existing, value) {
				return nil, stop("contract", "CONTRACT_CONFLICT", "WSDL binding definitions conflict")
			}
			bindings[name] = value
		}
		for name, value := range parsed.PortTypes {
			if existing, ok := portTypes[name]; ok && !reflect.DeepEqual(existing, value) {
				return nil, stop("contract", "CONTRACT_CONFLICT", "WSDL portType definitions conflict")
			}
			portTypes[name] = value
		}
		for name, value := range parsed.Messages {
			if existing, ok := messages[name]; ok && !reflect.DeepEqual(existing, value) {
				return nil, stop("contract", "CONTRACT_CONFLICT", "WSDL message definitions conflict")
			}
			messages[name] = value
		}
		for _, component := range parsed.XSDComponents {
			if component.Namespace != "" {
				targets = append(targets, sanitizeValue(component.Namespace, manifest, key))
			}
			componentKey := xsdComponentKey(component)
			if component.Redefines {
				if component.ParentQName != "" || (component.Kind != "complexType" && component.Kind != "simpleType") {
					return nil, stop("contract", "CONTRACT_CONFLICT", "unsupported XSD redefine component")
				}
				redefinedComponents[componentKey] = true
			} else {
				originalComponents[componentKey] = true
			}
			if existing, ok := componentIndex[componentKey]; ok {
				if reflect.DeepEqual(existing, component) {
					continue
				}
				if existing.Redefines == component.Redefines {
					return nil, stop("contract", "CONTRACT_CONFLICT", "XSD component definitions conflict")
				}
				if component.Redefines {
					componentIndex[componentKey] = component
				}
				continue
			}
			componentIndex[componentKey] = component
		}
	}
	for componentKey := range redefinedComponents {
		if !originalComponents[componentKey] {
			return nil, stop("contract", "CONTRACT_CONFLICT", "XSD redefine target did not resolve")
		}
	}
	for _, component := range componentIndex {
		if component.ParentQName == "" {
			symbol := qname(component.Namespace, component.Name)
			if component.Kind == "element" {
				xsdElements[symbol] = true
			}
			if component.Kind == "complexType" || component.Kind == "simpleType" {
				xsdTypes[symbol] = true
			}
		}
	}
	for _, component := range componentIndex {
		if component.Type != "" && !xsdTypes[component.Type] && !strings.HasPrefix(component.Type, "{"+xsdNamespace+"}") {
			return nil, stop("contract", "CONTRACT_MISMATCH", "XSD component type did not resolve in the closure")
		}
		components = append(components, sanitizeComponents([]XSDComponent{component}, manifest, key)[0])
	}

	ports, found := services[manifest.ExpectedServiceQName]
	if !found {
		return nil, stop("contract", "CONTRACT_MISMATCH", "expected WSDL service was not found")
	}
	var port portInfo
	for _, candidate := range ports {
		if candidate.Name == manifest.ExpectedPortQName {
			port = candidate
			break
		}
	}
	if port.Name == "" || port.BindingQName != manifest.ExpectedBindingQName {
		return nil, stop("contract", "CONTRACT_MISMATCH", "expected WSDL port or binding was not found")
	}
	binding, ok := bindings[manifest.ExpectedBindingQName]
	if !ok {
		return nil, stop("contract", "CONTRACT_MISMATCH", "expected WSDL binding was not found")
	}
	action, ok := binding.Actions[manifest.ExpectedOperation]
	if !ok || action != manifest.ExpectedSOAPAction {
		return nil, stop("contract", "CONTRACT_MISMATCH", "expected WSDL operation or SOAP action did not match")
	}
	operations := portTypes[binding.TypeQName]
	operation, ok := operations[manifest.ExpectedOperation]
	if !ok || operation.Input == "" {
		return nil, stop("contract", "CONTRACT_MISMATCH", "expected WSDL message exchange was not found")
	}
	soapVersion := binding.SOAPVersion
	if port.SOAPVersion != "" && soapVersion != port.SOAPVersion {
		return nil, stop("contract", "CONTRACT_MISMATCH", "SOAP binding versions disagree")
	}
	output := (*string)(nil)
	exchange := "one-way"
	if operation.Output != "" {
		value := sanitizeValue(operation.Output, manifest, key)
		output = &value
		exchange = "request-response"
	}
	referencedMessages := append([]string{operation.Input}, operation.Faults...)
	if operation.Output != "" {
		referencedMessages = append(referencedMessages, operation.Output)
	}
	messageSummaries := make([]MessageSummary, 0, len(referencedMessages))
	for _, messageQName := range uniqueSorted(referencedMessages) {
		if ctx.Err() != nil {
			return nil, stop("timeout", "WHOLE_ACTION_TIMEOUT", "whole-action timeout reached")
		}
		parts, ok := messages[messageQName]
		if !ok {
			return nil, stop("contract", "CONTRACT_MISMATCH", "referenced WSDL message definition was not found")
		}
		sanitizedParts := make([]MessagePart, 0, len(parts))
		for _, part := range parts {
			if (part.ElementQName == "") == (part.TypeQName == "") {
				return nil, stop("contract", "CONTRACT_MISMATCH", "WSDL message part must resolve exactly one element or type")
			}
			if part.ElementQName != "" && !xsdElements[part.ElementQName] {
				return nil, stop("contract", "CONTRACT_MISMATCH", "WSDL message part element did not resolve in the XSD closure")
			}
			if part.TypeQName != "" && !xsdTypes[part.TypeQName] && !strings.HasPrefix(part.TypeQName, "{"+xsdNamespace+"}") {
				return nil, stop("contract", "CONTRACT_MISMATCH", "WSDL message part type did not resolve in the XSD closure")
			}
			sanitizedParts = append(sanitizedParts, MessagePart{Name: sanitizeValue(part.Name, manifest, key), ElementQName: sanitizeValue(part.ElementQName, manifest, key), TypeQName: sanitizeValue(part.TypeQName, manifest, key)})
		}
		messageSummaries = append(messageSummaries, MessageSummary{QName: sanitizeValue(messageQName, manifest, key), Parts: sanitizedParts})
	}
	sort.Slice(components, func(i, j int) bool {
		if components[i].Namespace != components[j].Namespace {
			return components[i].Namespace < components[j].Namespace
		}
		if components[i].ParentQName != components[j].ParentQName {
			return components[i].ParentQName < components[j].ParentQName
		}
		if components[i].Order != components[j].Order {
			return components[i].Order < components[j].Order
		}
		if components[i].Name != components[j].Name {
			return components[i].Name < components[j].Name
		}
		return components[i].Kind < components[j].Kind
	})
	actionDigest := sha256.Sum256([]byte(action))
	return &ContractSummary{
		WSDLVersion: "1.1", TargetNamespaces: uniqueSorted(targets),
		ServiceQName: sanitizeValue(manifest.ExpectedServiceQName, manifest, key), PortQName: sanitizeValue(port.Name, manifest, key), BindingQName: sanitizeValue(port.BindingQName, manifest, key),
		SOAPVersion: soapVersion, Operation: manifest.ExpectedOperation, InputMessageQName: sanitizeValue(operation.Input, manifest, key), OutputMessageQName: output,
		FaultMessageQNames: uniqueSorted(sanitizeStrings(operation.Faults, manifest, key)), MessageExchange: exchange,
		SOAPActionSHA256: hex.EncodeToString(actionDigest[:]), SOAPActionMatchesSealedExpected: true,
		Messages: messageSummaries, XSDComponents: components, PolicyAssertionQNames: uniqueSorted(assertions),
	}, nil
}

func sortedDocumentURIs(documents map[string]*rawDocument) []string {
	result := make([]string, 0, len(documents))
	for uri := range documents {
		result = append(result, uri)
	}
	sort.Strings(result)
	return result
}

func xsdComponentKey(component XSDComponent) string {
	if component.ParentQName == "" {
		return component.Namespace + "\x00" + component.Kind + "\x00" + component.Name
	}
	return component.ParentQName + "\x00" + component.Kind + "\x00" + fmt.Sprintf("%08d", component.Order)
}

func resolveFetchURI(raw string, base *url.URL, manifest Manifest) (resolvedURI, error) {
	empty := resolvedURI{}
	if hasForbiddenURIText(raw) {
		return empty, fmt.Errorf("forbidden URI text")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return empty, err
	}
	if err := validateEscapedPath(parsed.EscapedPath()); err != nil {
		return empty, err
	}
	if base != nil {
		parsed = base.ResolveReference(parsed)
	}
	parsed.Fragment = ""
	if parsed.User != nil || parsed.Host == "" || canonicalOrigin(parsed) != manifest.AllowedOrigin {
		return empty, fmt.Errorf("origin mismatch")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())) {
		return empty, fmt.Errorf("scheme forbidden")
	}
	if err := validateEscapedPath(parsed.EscapedPath()); err != nil {
		return empty, fmt.Errorf("path forbidden")
	}
	if err := validateRawQuery(parsed.RawQuery, manifest.SAPClient); err != nil {
		return empty, err
	}
	requestURI := parsed.String()
	keyURI := *parsed
	canonicalPath, err := canonicalPercentEncoding(parsed.EscapedPath())
	if err != nil {
		return empty, fmt.Errorf("path forbidden")
	}
	decodedPath, err := url.PathUnescape(canonicalPath)
	if err != nil {
		return empty, fmt.Errorf("path forbidden")
	}
	if decodedPath == "" {
		decodedPath = "/"
		canonicalPath = "/"
	}
	keyURI.Path = decodedPath
	keyURI.RawPath = canonicalPath
	sealedOrigin, err := url.Parse(manifest.AllowedOrigin)
	if err != nil {
		return empty, fmt.Errorf("origin invalid")
	}
	keyURI.Scheme = sealedOrigin.Scheme
	keyURI.Host = sealedOrigin.Host
	return resolvedURI{RequestURI: requestURI, NormalizedKey: keyURI.String()}, nil
}

func resolveReference(baseURI, raw string, manifest Manifest) (resolvedURI, string, bool, error) {
	empty := resolvedURI{}
	if hasForbiddenURIText(raw) {
		return empty, "", false, fmt.Errorf("forbidden reference")
	}
	ref, err := url.Parse(raw)
	if err != nil {
		return empty, "", false, err
	}
	fragment := ref.Fragment
	if ref.Path == "" && ref.RawQuery == "" && ref.Host == "" && fragment != "" {
		base, err := resolveFetchURI(baseURI, nil, manifest)
		return base, fragment, true, err
	}
	base, err := url.Parse(baseURI)
	if err != nil {
		return empty, "", false, err
	}
	target, err := resolveFetchURI(raw, base, manifest)
	return target, fragment, false, err
}

func hasForbiddenURIText(raw string) bool {
	for _, r := range raw {
		if r < 0x20 || r == 0x7f || r == '\\' {
			return true
		}
	}
	return strings.Contains(raw, "@") && strings.Contains(raw, "://")
}

func containsDotTraversal(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

func validateEscapedPath(path string) error {
	decoded, err := url.PathUnescape(path)
	if err != nil || strings.Contains(decoded, "\\") || containsDotTraversal(decoded) {
		return fmt.Errorf("path forbidden")
	}
	for _, r := range decoded {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("path forbidden")
		}
	}
	return nil
}

func validateRawQuery(rawQuery, sapClient string) error {
	decoded, err := url.QueryUnescape(rawQuery)
	if err != nil {
		return fmt.Errorf("query forbidden")
	}
	for _, character := range decoded {
		if character < 0x20 || character == 0x7f || character == '\\' {
			return fmt.Errorf("query forbidden")
		}
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return fmt.Errorf("query forbidden")
	}
	clients, ok := values["sap-client"]
	if !ok || len(clients) != 1 || clients[0] != sapClient {
		return fmt.Errorf("sealed SAP client mismatch")
	}
	return nil
}

func canonicalPercentEncoding(value string) (string, error) {
	const upperHex = "0123456789ABCDEF"
	var canonical strings.Builder
	canonical.Grow(len(value))
	for index := 0; index < len(value); index++ {
		if value[index] != '%' {
			canonical.WriteByte(value[index])
			continue
		}
		if index+2 >= len(value) {
			return "", fmt.Errorf("truncated percent encoding")
		}
		high, ok := hexNibble(value[index+1])
		if !ok {
			return "", fmt.Errorf("invalid percent encoding")
		}
		low, ok := hexNibble(value[index+2])
		if !ok {
			return "", fmt.Errorf("invalid percent encoding")
		}
		decoded := high<<4 | low
		if isURIUnreserved(decoded) {
			canonical.WriteByte(decoded)
		} else {
			canonical.WriteByte('%')
			canonical.WriteByte(upperHex[decoded>>4])
			canonical.WriteByte(upperHex[decoded&0x0f])
		}
		index += 2
	}
	return canonical.String(), nil
}

func hexNibble(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}

func isURIUnreserved(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '-' || value == '.' || value == '_' || value == '~'
}

func documentID(key []byte, uri string) string { return "doc-" + hmacDigest(key, uri)[:32] }

func stop(phase, code, message string) *stopError {
	return &stopError{Phase: phase, Code: code, Message: message}
}
func isPolicyNamespace(namespace string) bool {
	return namespace == wsp15Namespace || namespace == wsp12Namespace
}
func attribute(value xml.StartElement, local string) string {
	for _, attr := range value.Attr {
		if attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}
func attributeOr(value xml.StartElement, local, fallback string) string {
	if got := attribute(value, local); got != "" {
		return got
	}
	return fallback
}
func qname(namespace, local string) string {
	if local == "" {
		return ""
	}
	return "{" + namespace + "}" + local
}
func expandQName(raw string, namespaces map[string]string) string {
	if raw == "" {
		return ""
	}
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) == 2 {
		return qname(namespaces[parts[0]], parts[1])
	}
	return qname(namespaces[""], raw)
}
func expandedName(name xml.Name) string { return qname(name.Space, name.Local) }
func copyNamespaces(source map[string]string) map[string]string {
	target := make(map[string]string, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}
func nearestComponent(active map[int]int, depth int) (int, bool) {
	bestDepth := -1
	index := 0
	for candidateDepth, candidate := range active {
		if candidateDepth < depth && candidateDepth > bestDepth {
			bestDepth = candidateDepth
			index = candidate
		}
	}
	return index, bestDepth >= 0
}
func nearestString(active map[int]string, depth int) (string, bool) {
	bestDepth := -1
	value := ""
	for candidateDepth, candidate := range active {
		if candidateDepth <= depth && candidateDepth > bestDepth {
			bestDepth = candidateDepth
			value = candidate
		}
	}
	return value, bestDepth >= 0
}
func isFacet(local string) bool {
	switch local {
	case "length", "minLength", "maxLength", "pattern", "enumeration", "minInclusive", "maxInclusive", "minExclusive", "maxExclusive", "totalDigits", "fractionDigits", "whiteSpace":
		return true
	}
	return false
}

func isXSD11OnlyElement(local string) bool {
	switch local {
	case "override", "assert", "assertion", "alternative", "openContent", "defaultOpenContent", "explicitTimezone":
		return true
	default:
		return false
	}
}

func canonicalEdges(edges []Edge) []Edge {
	sort.Slice(edges, func(i, j int) bool {
		left := edges[i].FromDocumentID + "\x00" + edges[i].ToDocumentID + "\x00" + edges[i].Relation
		right := edges[j].FromDocumentID + "\x00" + edges[j].ToDocumentID + "\x00" + edges[j].Relation
		return left < right
	})
	result := edges[:0]
	var previous string
	for _, edge := range edges {
		key := edge.FromDocumentID + "\x00" + edge.ToDocumentID + "\x00" + edge.Relation
		if len(result) == 0 || key != previous {
			result = append(result, edge)
			previous = key
		}
	}
	return result
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
func sanitizeStrings(values []string, manifest Manifest, key []byte) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, sanitizeValue(value, manifest, key))
	}
	return result
}
func sanitizeComponents(values []XSDComponent, manifest Manifest, key []byte) []XSDComponent {
	result := make([]XSDComponent, 0, len(values))
	for _, value := range values {
		facets := make([]XSDFacet, 0, len(value.Facets))
		for _, facet := range value.Facets {
			facets = append(facets, XSDFacet{Name: facet.Name, Value: sanitizeValue(facet.Value, manifest, key)})
		}
		result = append(result, XSDComponent{Namespace: sanitizeValue(value.Namespace, manifest, key), ParentQName: sanitizeValue(value.ParentQName, manifest, key), Name: sanitizeValue(value.Name, manifest, key), Kind: value.Kind, Order: value.Order, Type: sanitizeValue(value.Type, manifest, key), MinOccurs: value.MinOccurs, MaxOccurs: value.MaxOccurs, Facets: facets})
	}
	return result
}
func sanitizeValue(value string, manifest Manifest, key []byte) string {
	for _, private := range []string{manifest.RootURL, manifest.AllowedOrigin, manifest.ExpectedSOAPAction} {
		if private != "" && strings.Contains(value, private) {
			value = strings.ReplaceAll(value, private, "private-hmac:"+hmacDigest(key, private))
		}
	}
	return value
}

func sanitizedDocumentSummary(documentID string, parsed parsedDocument, manifest Manifest, key []byte) map[string]interface{} {
	services := map[string][]portInfo{}
	for name, ports := range parsed.Services {
		safePorts := make([]portInfo, 0, len(ports))
		for _, port := range ports {
			safePorts = append(safePorts, portInfo{Name: sanitizeValue(port.Name, manifest, key), BindingQName: sanitizeValue(port.BindingQName, manifest, key), SOAPVersion: port.SOAPVersion})
		}
		services[sanitizeValue(name, manifest, key)] = safePorts
	}
	bindings := map[string]interface{}{}
	for name, binding := range parsed.Bindings {
		actions := map[string]interface{}{}
		for operation, action := range binding.Actions {
			digest := sha256.Sum256([]byte(action))
			actions[operation] = map[string]interface{}{"soap_action_sha256": hex.EncodeToString(digest[:]), "matches_sealed_expected": action == manifest.ExpectedSOAPAction}
		}
		bindings[sanitizeValue(name, manifest, key)] = map[string]interface{}{"type_qname": sanitizeValue(binding.TypeQName, manifest, key), "soap_version": binding.SOAPVersion, "operations": actions}
	}
	portTypes := map[string]interface{}{}
	for name, operations := range parsed.PortTypes {
		safeOperations := map[string]interface{}{}
		for operationName, operation := range operations {
			safeOperations[operationName] = map[string]interface{}{"input": sanitizeValue(operation.Input, manifest, key), "output": sanitizeValue(operation.Output, manifest, key), "faults": sanitizeStrings(operation.Faults, manifest, key)}
		}
		portTypes[sanitizeValue(name, manifest, key)] = safeOperations
	}
	messages := map[string][]MessagePart{}
	for name, parts := range parsed.Messages {
		safeParts := make([]MessagePart, 0, len(parts))
		for _, part := range parts {
			safeParts = append(safeParts, MessagePart{Name: sanitizeValue(part.Name, manifest, key), ElementQName: sanitizeValue(part.ElementQName, manifest, key), TypeQName: sanitizeValue(part.TypeQName, manifest, key)})
		}
		messages[sanitizeValue(name, manifest, key)] = safeParts
	}
	return map[string]interface{}{
		"document_id":             documentID,
		"kind":                    parsed.Kind,
		"target_namespace":        sanitizeValue(parsed.TargetNamespace, manifest, key),
		"services":                services,
		"bindings":                bindings,
		"port_types":              portTypes,
		"messages":                messages,
		"xsd_components":          sanitizeComponents(parsed.XSDComponents, manifest, key),
		"policy_assertion_qnames": sanitizeStrings(parsed.PolicyAssertions, manifest, key),
	}
}

func bundleLeak(manifest Manifest, credentials Credentials, bundle *Bundle) bool {
	encoded, err := json.Marshal(bundle)
	if err != nil {
		return true
	}
	text := string(encoded)
	for _, forbidden := range []string{manifest.RootURL, manifest.AllowedOrigin, credentials.Username, credentials.Password, manifest.ExpectedSOAPAction, "sap-client=" + manifest.SAPClient} {
		if forbidden != "" && strings.Contains(text, forbidden) {
			return true
		}
	}
	return false
}
