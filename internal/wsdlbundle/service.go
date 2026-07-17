package wsdlbundle

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const zeroDigest = "0000000000000000000000000000000000000000000000000000000000000000"

type PermitLedger interface {
	Consume(input Input) error
}

type ServiceConfig struct {
	ActiveSystemID   string
	Manifest         Manifest
	ManifestSHA256   string
	Ledger           PermitLedger
	Credentials      Credentials
	EvidenceKey      []byte
	TransportFactory func(context.Context, Manifest) (http.RoundTripper, error)
	AttemptID        func() (string, error)
}

type Service struct {
	config ServiceConfig
}

func NewService(config ServiceConfig) (*Service, error) {
	if err := validateManifest(config.Manifest, false); err != nil {
		return nil, err
	}
	computed, err := ManifestSHA256(config.Manifest)
	if err != nil || computed != config.ManifestSHA256 {
		return nil, fmt.Errorf("MANIFEST_DIGEST_MISMATCH")
	}
	if config.ActiveSystemID != SystemID || config.Ledger == nil || config.Credentials.Username == "" || config.Credentials.Password == "" || len(config.EvidenceKey) < 32 {
		return nil, fmt.Errorf("SERVICE_CONFIG_INVALID")
	}
	if config.AttemptID == nil {
		config.AttemptID = newUUIDv4
	}
	if config.TransportFactory == nil {
		config.TransportFactory = newStrictTransport
	}
	return &Service{config: config}, nil
}

func (s *Service) Fetch(ctx context.Context, args map[string]interface{}) (Result, error) {
	input, err := ParseInput(args)
	if err != nil {
		return Result{}, err
	}
	attemptID, err := s.config.AttemptID()
	if err != nil {
		return Result{}, fmt.Errorf("ATTEMPT_ID_UNAVAILABLE")
	}
	return s.fetchInput(ctx, input, attemptID), nil
}

func (s *Service) fetchInput(ctx context.Context, input Input, attemptID string) Result {
	result := baseResult(input, attemptID)
	if s.config.ActiveSystemID != SystemID {
		return hardStopResult(result, "identity", "IDENTITY_MISMATCH", "active runtime identity is not the sealed GPI system")
	}
	if input.RequestManifestSHA256 != s.config.ManifestSHA256 || input.SystemID != s.config.Manifest.SystemID || input.ContractID != s.config.Manifest.ContractID {
		return hardStopResult(result, "manifest", "MANIFEST_MISMATCH", "sealed request manifest mismatch")
	}
	result.Identity = Identity{
		MatchesSealedManifest:       true,
		ConnectionBindingHMACSHA256: hmacDigest(s.config.EvidenceKey, s.config.Manifest.AllowedOrigin+"\n"+s.config.Manifest.SAPClient+"\n"+s.config.Manifest.SystemID),
	}
	if err := s.config.Ledger.Consume(input); err != nil {
		code := permitErrorCode(err)
		return hardStopResult(result, "permit", code, "one-shot permit rejected")
	}
	result.PermitConsumed = true

	actionCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.Manifest.Limits.WholeActionTimeoutMS)*time.Millisecond)
	defer cancel()
	if actionCtx.Err() != nil {
		return hardStopResult(result, "timeout", "WHOLE_ACTION_TIMEOUT", "whole-action timeout reached")
	}
	roundTripper, err := s.config.TransportFactory(actionCtx, s.config.Manifest)
	if err != nil {
		return hardStopResult(result, "transport", "TRANSPORT_INIT_FAILED", "strict transport initialization failed")
	}
	fetched, stop := fetchClosure(actionCtx, s.config.Manifest, s.config.Credentials, s.config.EvidenceKey, roundTripper)
	result.NetworkGetsStarted = fetched.NetworkGetsStarted
	result.Identity.TLSPeerSHA256 = fetched.TLSPeerSHA256
	if stop != nil {
		return hardStopResult(result, stop.Phase, stop.Code, stop.Message)
	}
	if actionCtx.Err() != nil {
		return hardStopResult(result, "timeout", "WHOLE_ACTION_TIMEOUT", "whole-action timeout reached")
	}
	result.Bundle = fetched.Bundle
	result.Outcome = "COMPLETE"
	placeholderDigest := zeroDigest
	result.EvidenceManifestSHA256 = &placeholderDigest
	if err := validateResult(result); err != nil {
		result.Bundle = nil
		result.EvidenceManifestSHA256 = nil
		return hardStopResult(result, "output", "OUTPUT_SCHEMA_INVALID", "result failed its declared schema invariants")
	}
	digest, err := publishEvidence(actionCtx, s.config.Manifest, result.AttemptID, fetched.Bundle)
	if err != nil {
		result.Bundle = nil
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return hardStopResult(result, "timeout", "WHOLE_ACTION_TIMEOUT", "whole-action timeout reached")
		}
		return hardStopResult(result, "evidence", "ATOMIC_PUBLISH_FAILED", "sanitized evidence was not published")
	}
	result.EvidenceManifestSHA256 = &digest
	if err := validateResult(result); err != nil {
		result.Bundle = nil
		result.EvidenceManifestSHA256 = nil
		return hardStopResult(result, "output", "OUTPUT_SCHEMA_INVALID", "result failed its declared schema invariants")
	}
	return result
}

func baseResult(input Input, attemptID string) Result {
	return Result{
		SchemaVersion:         "1.0",
		Outcome:               "HARD_STOP",
		AttemptID:             attemptID,
		SystemID:              input.SystemID,
		ContractID:            input.ContractID,
		RequestManifestSHA256: input.RequestManifestSHA256,
		Identity:              Identity{ConnectionBindingHMACSHA256: zeroDigest},
	}
}

func hardStopResult(result Result, phase, code, message string) Result {
	result.Outcome = "HARD_STOP"
	result.Bundle = nil
	result.EvidenceManifestSHA256 = nil
	result.HardStop = &HardStop{Phase: phase, Code: code, Message: message}
	return result
}

func permitErrorCode(err error) string {
	var permitErr *PermitError
	if errors.As(err, &permitErr) && stopCodePattern.MatchString(permitErr.Code) {
		return permitErr.Code
	}
	return "PERMIT_REJECTED"
}

func validateResult(result Result) error {
	if result.SchemaVersion != "1.0" || !uuidV4Pattern.MatchString(result.AttemptID) || result.SystemID != SystemID || result.ContractID != ContractID || !lowerSHA256Pattern.MatchString(result.RequestManifestSHA256) || !lowerSHA256Pattern.MatchString(result.Identity.ConnectionBindingHMACSHA256) || result.NetworkGetsStarted < 0 || result.NetworkGetsStarted > 64 {
		return fmt.Errorf("base result invariant failed")
	}
	if result.Identity.TLSPeerSHA256 != nil && !lowerSHA256Pattern.MatchString(*result.Identity.TLSPeerSHA256) {
		return fmt.Errorf("TLS identity invariant failed")
	}
	if result.Outcome == "COMPLETE" {
		if !result.Identity.MatchesSealedManifest || !result.PermitConsumed || result.NetworkGetsStarted < 1 || result.Bundle == nil || result.HardStop != nil || result.EvidenceManifestSHA256 == nil || !lowerSHA256Pattern.MatchString(*result.EvidenceManifestSHA256) || validateBundle(result.Bundle) != nil {
			return fmt.Errorf("complete result invariant failed")
		}
		return nil
	}
	if result.Outcome != "HARD_STOP" || result.Bundle != nil || result.HardStop == nil || result.EvidenceManifestSHA256 != nil || result.HardStop.Phase == "" || result.HardStop.Message == "" || !stopCodePattern.MatchString(result.HardStop.Code) {
		return fmt.Errorf("hard-stop result invariant failed")
	}
	return nil
}

func validateBundle(bundle *Bundle) error {
	if bundle == nil || !bundle.Complete || !documentIDPattern.MatchString(bundle.RootDocumentID) || !lowerSHA256Pattern.MatchString(bundle.BundleSHA256) || len(bundle.Documents) < 1 || len(bundle.Documents) > 64 || len(bundle.Edges) > 256 {
		return fmt.Errorf("bundle invariant failed")
	}
	for _, document := range bundle.Documents {
		if !documentIDPattern.MatchString(document.DocumentID) || (document.Kind != "WSDL11" && document.Kind != "XSD10" && document.Kind != "WS_POLICY") || document.ByteCount < 1 || document.ByteCount > 4*1024*1024 || !lowerSHA256Pattern.MatchString(document.RawSHA256) || !lowerSHA256Pattern.MatchString(document.SanitizedSHA256) || !isXMLMediaType(document.MediaType) {
			return fmt.Errorf("document invariant failed")
		}
	}
	for _, edge := range bundle.Edges {
		if !documentIDPattern.MatchString(edge.FromDocumentID) || !documentIDPattern.MatchString(edge.ToDocumentID) || (edge.Relation != "wsdl_import" && edge.Relation != "xsd_import" && edge.Relation != "xsd_include" && edge.Relation != "xsd_redefine" && edge.Relation != "policy_reference") {
			return fmt.Errorf("edge invariant failed")
		}
	}
	contract := bundle.Contract
	if contract.WSDLVersion != "1.1" || contract.ServiceQName == "" || contract.PortQName == "" || contract.BindingQName == "" || (contract.SOAPVersion != "1.1" && contract.SOAPVersion != "1.2") || contract.Operation == "" || contract.InputMessageQName == "" || (contract.MessageExchange != "request-response" && contract.MessageExchange != "one-way") || !lowerSHA256Pattern.MatchString(contract.SOAPActionSHA256) || !contract.SOAPActionMatchesSealedExpected {
		return fmt.Errorf("contract invariant failed")
	}
	if contract.MessageExchange == "request-response" && contract.OutputMessageQName == nil {
		return fmt.Errorf("message exchange invariant failed")
	}
	if contract.MessageExchange == "one-way" && contract.OutputMessageQName != nil {
		return fmt.Errorf("message exchange invariant failed")
	}
	for _, message := range contract.Messages {
		if message.QName == "" {
			return fmt.Errorf("message invariant failed")
		}
		for _, part := range message.Parts {
			if part.Name == "" || (part.ElementQName == "" && part.TypeQName == "") {
				return fmt.Errorf("message part invariant failed")
			}
		}
	}
	componentByID := make(map[string]XSDComponent, len(contract.XSDComponents))
	for _, component := range contract.XSDComponents {
		if _, exists := componentByID[component.ComponentID]; exists {
			return fmt.Errorf("duplicate XSD component identity")
		}
		componentByID[component.ComponentID] = component
		if component.ComponentID == "" || !validXSDComponentKind(component.Kind) || component.Order < 0 || component.MinOccurs == "" || component.MaxOccurs == "" || component.TypeReferences == nil || component.Facets == nil {
			return fmt.Errorf("XSD component invariant failed")
		}
		if component.ParentID == "" {
			if component.Anonymous || component.Name == "" || component.ComponentID != globalXSDComponentID(component.Namespace, component.Kind, component.Name) {
				return fmt.Errorf("global XSD component invariant failed")
			}
		} else if component.ComponentID != localXSDComponentID(component.ParentID, component.Kind, component.Order) {
			return fmt.Errorf("local XSD component invariant failed")
		}
		if component.Anonymous {
			if component.Name != "" || (component.Kind != "complexType" && component.Kind != "simpleType") || component.ParentID == "" {
				return fmt.Errorf("anonymous XSD component invariant failed")
			}
		} else if component.Name == "" {
			return fmt.Errorf("named XSD component invariant failed")
		}
		if component.Nillable && component.Kind != "element" {
			return fmt.Errorf("XSD nillability invariant failed")
		}
		if component.InlineTypeID != "" && ((component.Kind != "element" && component.Kind != "attribute") || component.Type != "" || component.RefQName != "" || !strings.HasPrefix(component.InlineTypeID, component.ComponentID+"/")) {
			return fmt.Errorf("inline XSD type invariant failed")
		}
		switch component.Derivation {
		case "", "restriction", "extension", "list", "union":
		default:
			return fmt.Errorf("XSD derivation invariant failed")
		}
		for _, facet := range component.Facets {
			if facet.Name == "" {
				return fmt.Errorf("XSD facet invariant failed")
			}
		}
	}
	for _, component := range contract.XSDComponents {
		if component.ParentID != "" {
			parent, ok := componentByID[component.ParentID]
			if !ok {
				return fmt.Errorf("XSD component parent invariant failed")
			}
			if component.Anonymous {
				inlineOwner := parent.Kind == "element" || parent.Kind == "attribute"
				nestedSimpleOwner := component.Kind == "simpleType" && parent.Kind == "simpleType" && (parent.Derivation == "restriction" || parent.Derivation == "list" || parent.Derivation == "union")
				if (!inlineOwner && !nestedSimpleOwner) || (inlineOwner && parent.InlineTypeID != component.ComponentID) {
					return fmt.Errorf("anonymous XSD owner invariant failed")
				}
			}
		}
		if component.InlineTypeID != "" {
			inlineType, ok := componentByID[component.InlineTypeID]
			if !ok || !inlineType.Anonymous || inlineType.ParentID != component.ComponentID || (inlineType.Kind != "complexType" && inlineType.Kind != "simpleType") {
				return fmt.Errorf("inline XSD type target invariant failed")
			}
		}
	}
	return nil
}

func validXSDComponentKind(kind string) bool {
	switch kind {
	case "element", "complexType", "simpleType", "group", "attributeGroup", "attribute":
		return true
	default:
		return false
	}
}

func hmacDigest(key []byte, value string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func newUUIDv4() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}

func FetchFromEnvironment(ctx context.Context, args map[string]interface{}, activeSystemID string) (Result, error) {
	input, err := ParseInput(args)
	if err != nil {
		return Result{}, err
	}
	attemptID, err := newUUIDv4()
	if err != nil {
		return Result{}, fmt.Errorf("ATTEMPT_ID_UNAVAILABLE")
	}
	if activeSystemID != SystemID {
		result := baseResult(input, attemptID)
		return hardStopResult(result, "identity", "IDENTITY_MISMATCH", "active runtime identity is not the sealed GPI system"), nil
	}
	service, err := loadEnvironmentService(attemptID, activeSystemID)
	if err != nil {
		result := baseResult(input, attemptID)
		return hardStopResult(result, "configuration", "SEALED_CONFIG_UNAVAILABLE", "sealed WSDL fetch configuration is unavailable"), nil
	}
	return service.fetchInput(ctx, input, attemptID), nil
}

func loadEnvironmentService(attemptID, activeSystemID string) (*Service, error) {
	manifestPath := strings.TrimSpace(os.Getenv("SAP_WSDL_BUNDLE_MANIFEST_FILE"))
	if manifestPath == "" {
		return nil, fmt.Errorf("manifest path missing")
	}
	var manifest Manifest
	if err := readPrivateJSON(manifestPath, &manifest, 128*1024); err != nil {
		return nil, err
	}
	if err := validateManifest(manifest, true); err != nil {
		return nil, err
	}
	manifestSHA, err := ManifestSHA256(manifest)
	if err != nil {
		return nil, err
	}
	var credentials Credentials
	if err := readPrivateJSON(manifest.CredentialFile, &credentials, 16*1024); err != nil {
		return nil, err
	}
	keyInfo, err := os.Lstat(manifest.EvidenceHMACKeyFile)
	if err != nil || !keyInfo.Mode().IsRegular() || keyInfo.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("evidence key unsafe")
	}
	key, err := os.ReadFile(manifest.EvidenceHMACKeyFile)
	if err != nil || len(key) < 32 {
		return nil, fmt.Errorf("evidence key unavailable")
	}
	binarySHA, err := executableSHA256()
	if err != nil {
		return nil, err
	}
	return NewService(ServiceConfig{
		ActiveSystemID: activeSystemID,
		Manifest:       manifest,
		ManifestSHA256: manifestSHA,
		Ledger:         &FilePermitLedger{Dir: manifest.PermitDir, BinarySHA256: binarySHA},
		Credentials:    credentials,
		EvidenceKey:    key,
		AttemptID:      func() (string, error) { return attemptID, nil },
	})
}

func executableSHA256() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func publishEvidence(ctx context.Context, manifest Manifest, attemptID string, bundle *Bundle) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if bundle == nil {
		return "", fmt.Errorf("bundle missing")
	}
	body, err := json.Marshal(bundle)
	if err != nil || int64(len(body)) > manifest.Limits.MaxEvidenceBytes {
		return "", fmt.Errorf("evidence size invalid")
	}
	if err := os.MkdirAll(manifest.EvidenceDir, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(manifest.EvidenceDir, 0o700); err != nil {
		return "", err
	}
	temp, err := os.CreateTemp(manifest.EvidenceDir, ".wsdl-evidence-*")
	if err != nil {
		return "", err
	}
	tempName := temp.Name()
	committed := false
	defer func() {
		_ = temp.Close()
		if !committed {
			_ = os.Remove(tempName)
		}
	}()
	if err := temp.Chmod(0o600); err != nil {
		return "", err
	}
	if _, err := temp.Write(body); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := temp.Sync(); err != nil {
		return "", err
	}
	if err := temp.Close(); err != nil {
		return "", err
	}
	finalName := filepath.Join(manifest.EvidenceDir, attemptID+".json")
	if err := os.Rename(tempName, finalName); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		_ = os.Remove(finalName)
		return "", err
	}
	dir, err := os.Open(manifest.EvidenceDir)
	if err != nil {
		_ = os.Remove(finalName)
		return "", err
	}
	if err := dir.Sync(); err != nil {
		_ = dir.Close()
		_ = os.Remove(finalName)
		return "", err
	}
	if err := dir.Close(); err != nil {
		_ = os.Remove(finalName)
		return "", err
	}
	if err := ctx.Err(); err != nil {
		_ = os.Remove(finalName)
		return "", err
	}
	committed = true
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}
