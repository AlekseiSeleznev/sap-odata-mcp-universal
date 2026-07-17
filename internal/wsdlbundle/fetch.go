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
	URI   string
	Depth int
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
	Raw      string
	Relation string
}

type parsedDocument struct {
	Kind             string
	TargetNamespace  string
	References       []reference
	IDs              map[string]bool
	PortTypes        map[string]map[string]operationInfo
	Bindings         map[string]bindingInfo
	Services         map[string][]portInfo
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
	root, err := normalizedFetchURI(manifest.RootURL, nil, manifest)
	if err != nil {
		return result, stop("uri", "URI_POLICY_VIOLATION", "sealed root violates URI policy")
	}
	queue := []queueItem{{URI: root, Depth: 0}}
	seen := map[string]bool{}
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
		if seen[item.URI] {
			continue
		}
		if item.Depth > manifest.Limits.MaxDepth {
			return result, stop("closure", "DEPTH_LIMIT", "recursive closure depth limit exceeded")
		}
		if len(seen) >= manifest.Limits.MaxDocuments {
			return result, stop("closure", "DOCUMENT_LIMIT", "document count limit exceeded")
		}
		seen[item.URI] = true
		result.NetworkGetsStarted++
		doc, tlsPeer, fetchStop := fetchDocument(ctx, item.URI, manifest, credentials, roundTripper)
		if fetchStop != nil {
			return result, fetchStop
		}
		if tlsPeer != nil {
			if result.TLSPeerSHA256 != nil && *result.TLSPeerSHA256 != *tlsPeer {
				return result, stop("identity", "TLS_IDENTITY_CHANGED", "TLS peer identity changed inside one action")
			}
			result.TLSPeerSHA256 = tlsPeer
		}
		totalBytes += int64(len(doc.Raw))
		if totalBytes > manifest.Limits.MaxTotalBytes {
			return result, stop("limits", "TOTAL_SIZE_LIMIT", "total raw byte limit exceeded")
		}
		doc.ID = documentID(evidenceKey, item.URI)
		documents[item.URI] = doc
		if fragments := requiredFragments[item.URI]; len(fragments) > 0 {
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
			target, fragment, sameDocument, err := resolveReference(item.URI, ref.Raw, manifest)
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
				if requiredFragments[target] == nil {
					requiredFragments[target] = map[string]bool{}
				}
				requiredFragments[target][fragment] = true
			}
			targetID := documentID(evidenceKey, target)
			edges = append(edges, Edge{FromDocumentID: doc.ID, ToDocumentID: targetID, Relation: ref.Relation})
			if !seen[target] {
				queue = append(queue, queueItem{URI: target, Depth: item.Depth + 1})
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

	contract, contractStop := buildContract(manifest, credentials, documents, evidenceKey)
	if contractStop != nil {
		return result, contractStop
	}
	docEvidence := make([]DocumentEvidence, 0, len(documents))
	for _, doc := range documents {
		rawDigest := sha256.Sum256(doc.Raw)
		sanitized := map[string]interface{}{
			"document_id":             doc.ID,
			"kind":                    doc.Kind,
			"target_namespace":        sanitizeValue(doc.Parsed.TargetNamespace, manifest, evidenceKey),
			"xsd_components":          sanitizeComponents(doc.Parsed.XSDComponents, manifest, evidenceKey),
			"policy_assertion_qnames": sanitizeStrings(doc.Parsed.PolicyAssertions, manifest, evidenceKey),
		}
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
	bundle := &Bundle{Complete: true, RootDocumentID: documentID(evidenceKey, root), Documents: docEvidence, Edges: edges, Contract: *contract}
	projection := struct {
		RootDocumentID string             `json:"root_document_id"`
		Documents      []DocumentEvidence `json:"documents"`
		Edges          []Edge             `json:"edges"`
		Contract       ContractSummary    `json:"contract"`
	}{bundle.RootDocumentID, bundle.Documents, bundle.Edges, bundle.Contract}
	canonical, err := json.Marshal(projection)
	if err != nil {
		return result, stop("digest", "DIGEST_FAILED", "bundle digest failed")
	}
	bundleDigest := sha256.Sum256(canonical)
	bundle.BundleSHA256 = hex.EncodeToString(bundleDigest[:])
	if leak := bundleLeak(manifest, credentials, bundle); leak {
		return result, stop("sanitize", "SANITIZATION_FAILED", "forbidden private value remained in evidence")
	}
	result.Bundle = bundle
	return result, nil
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
	parsed, parseStop := parseXMLDocument(body, manifest.Limits)
	if parseStop != nil {
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

func parseXMLDocument(body []byte, limits Limits) (parsedDocument, *stopError) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.Strict = true
	parsed := parsedDocument{IDs: map[string]bool{}, PortTypes: map[string]map[string]operationInfo{}, Bindings: map[string]bindingInfo{}, Services: map[string][]portInfo{}}
	namespaces := []map[string]string{{}}
	depth := 0
	tokens := 0
	var rootSeen bool
	var currentPortType, currentBinding, currentService string
	var currentPortTypeDepth, currentBindingDepth, currentServiceDepth int
	var currentPTOperation, currentBindingOperation string
	var currentPTOperationDepth, currentBindingOperationDepth int
	componentAtDepth := map[int]int{}
	policyDepth := 0

	for {
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
			if value.Name.Space == wsdl20Namespace || (value.Name.Space == xsdNamespace && value.Name.Local == "override") {
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
			if (value.Name.Space == wsdl11Namespace && value.Name.Local == "definitions") || (value.Name.Space == xsdNamespace && value.Name.Local == "schema") {
				if target := attribute(value, "targetNamespace"); target != "" {
					parsed.TargetNamespace = target
				}
			}
			if ref, relation, required := referenceAttribute(value); relation != "" {
				if required && strings.TrimSpace(ref) == "" {
					return parsed, stop("xml", "REFERENCE_MISSING", "required XML reference is missing")
				}
				if strings.TrimSpace(ref) != "" {
					parsed.References = append(parsed.References, reference{Raw: ref, Relation: relation})
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
				case "portType":
					currentPortType = qname(parsed.TargetNamespace, attribute(value, "name"))
					currentPortTypeDepth = depth
					parsed.PortTypes[currentPortType] = map[string]operationInfo{}
				case "binding":
					currentBinding = qname(parsed.TargetNamespace, attribute(value, "name"))
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
				if value.Name.Local == "element" || value.Name.Local == "complexType" || value.Name.Local == "simpleType" {
					if name := attribute(value, "name"); name != "" {
						component := XSDComponent{Namespace: parsed.TargetNamespace, Name: name, Kind: value.Name.Local, Type: expandQName(attribute(value, "type"), ns), MinOccurs: attributeOr(value, "minOccurs", "1"), MaxOccurs: attributeOr(value, "maxOccurs", "1"), Facets: map[string]string{}}
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
						component.Facets[value.Name.Local] = attribute(value, "value")
						parsed.XSDComponents[index] = component
					}
				}
			}
		case xml.EndElement:
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
			delete(componentAtDepth, depth)
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

func buildContract(manifest Manifest, credentials Credentials, documents map[string]*rawDocument, key []byte) (*ContractSummary, *stopError) {
	var selected parsedDocument
	found := false
	for _, doc := range documents {
		if _, ok := doc.Parsed.Services[manifest.ExpectedServiceQName]; ok {
			selected = doc.Parsed
			found = true
			break
		}
	}
	if !found {
		return nil, stop("contract", "CONTRACT_MISMATCH", "expected WSDL service was not found")
	}
	ports := selected.Services[manifest.ExpectedServiceQName]
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
	binding, ok := selected.Bindings[manifest.ExpectedBindingQName]
	if !ok {
		return nil, stop("contract", "CONTRACT_MISMATCH", "expected WSDL binding was not found")
	}
	action, ok := binding.Actions[manifest.ExpectedOperation]
	if !ok || action != manifest.ExpectedSOAPAction {
		return nil, stop("contract", "CONTRACT_MISMATCH", "expected WSDL operation or SOAP action did not match")
	}
	operations := selected.PortTypes[binding.TypeQName]
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
	targets := []string{}
	components := []XSDComponent{}
	assertions := []string{}
	for _, doc := range documents {
		if doc.Parsed.TargetNamespace != "" {
			targets = append(targets, sanitizeValue(doc.Parsed.TargetNamespace, manifest, key))
		}
		components = append(components, sanitizeComponents(doc.Parsed.XSDComponents, manifest, key)...)
		assertions = append(assertions, sanitizeStrings(doc.Parsed.PolicyAssertions, manifest, key)...)
	}
	sort.Slice(components, func(i, j int) bool {
		if components[i].Namespace != components[j].Namespace {
			return components[i].Namespace < components[j].Namespace
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
		XSDComponents: components, PolicyAssertionQNames: uniqueSorted(assertions),
	}, nil
}

func normalizedFetchURI(raw string, base *url.URL, manifest Manifest) (string, error) {
	if hasForbiddenURIText(raw) {
		return "", fmt.Errorf("forbidden URI text")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if base != nil {
		parsed = base.ResolveReference(parsed)
	}
	parsed.Fragment = ""
	if parsed.User != nil || parsed.Host == "" || canonicalOrigin(parsed) != manifest.AllowedOrigin {
		return "", fmt.Errorf("origin mismatch")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())) {
		return "", fmt.Errorf("scheme forbidden")
	}
	if decoded, err := url.PathUnescape(parsed.EscapedPath()); err != nil || strings.Contains(decoded, "\\") || containsDotTraversal(decoded) {
		return "", fmt.Errorf("path forbidden")
	}
	return parsed.String(), nil
}

func resolveReference(baseURI, raw string, manifest Manifest) (string, string, bool, error) {
	if hasForbiddenURIText(raw) {
		return "", "", false, fmt.Errorf("forbidden reference")
	}
	ref, err := url.Parse(raw)
	if err != nil {
		return "", "", false, err
	}
	fragment := ref.Fragment
	if ref.Path == "" && ref.RawQuery == "" && ref.Host == "" && fragment != "" {
		return baseURI, fragment, true, nil
	}
	base, err := url.Parse(baseURI)
	if err != nil {
		return "", "", false, err
	}
	target, err := normalizedFetchURI(raw, base, manifest)
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
func isFacet(local string) bool {
	switch local {
	case "length", "minLength", "maxLength", "pattern", "enumeration", "minInclusive", "maxInclusive", "minExclusive", "maxExclusive", "totalDigits", "fractionDigits", "whiteSpace":
		return true
	}
	return false
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
		facets := map[string]string{}
		for name, facet := range value.Facets {
			facets[name] = sanitizeValue(facet, manifest, key)
		}
		result = append(result, XSDComponent{Namespace: sanitizeValue(value.Namespace, manifest, key), Name: sanitizeValue(value.Name, manifest, key), Kind: value.Kind, Type: sanitizeValue(value.Type, manifest, key), MinOccurs: value.MinOccurs, MaxOccurs: value.MaxOccurs, Facets: facets})
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
