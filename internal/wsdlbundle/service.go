package wsdlbundle

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	Manifest         Manifest
	ManifestSHA256   string
	Ledger           PermitLedger
	Credentials      Credentials
	EvidenceKey      []byte
	TransportFactory func(context.Context, Manifest) (http.RoundTripper, error)
	AttemptID        func() string
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
	if config.Ledger == nil || config.Credentials.Username == "" || config.Credentials.Password == "" || len(config.EvidenceKey) < 32 {
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
	return s.fetchInput(ctx, input), nil
}

func (s *Service) fetchInput(ctx context.Context, input Input) Result {
	result := baseResult(input, s.config.AttemptID())
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
	result.Bundle = fetched.Bundle
	digest, err := publishEvidence(s.config.Manifest, result.AttemptID, fetched.Bundle)
	if err != nil {
		result.Bundle = nil
		return hardStopResult(result, "evidence", "ATOMIC_PUBLISH_FAILED", "sanitized evidence was not published")
	}
	result.Outcome = "COMPLETE"
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
	for _, code := range []string{"PERMIT_REPLAY", "PERMIT_MISMATCH", "PERMIT_UNAVAILABLE", "PERMIT_FILE_UNSAFE", "PERMIT_INVALID", "PERMIT_CONFIG_INVALID", "PERMIT_CONSUME_FAILED"} {
		if strings.Contains(err.Error(), code) {
			return code
		}
	}
	return "PERMIT_REJECTED"
}

func validateResult(result Result) error {
	if result.SchemaVersion != "1.0" || !uuidV4Pattern.MatchString(result.AttemptID) || result.SystemID != SystemID || result.ContractID != ContractID || !lowerSHA256Pattern.MatchString(result.RequestManifestSHA256) || !lowerSHA256Pattern.MatchString(result.Identity.ConnectionBindingHMACSHA256) || result.NetworkGetsStarted < 0 || result.NetworkGetsStarted > 64 {
		return fmt.Errorf("base result invariant failed")
	}
	if result.Outcome == "COMPLETE" {
		if !result.PermitConsumed || result.NetworkGetsStarted < 1 || result.Bundle == nil || result.HardStop != nil || result.EvidenceManifestSHA256 == nil || !lowerSHA256Pattern.MatchString(*result.EvidenceManifestSHA256) {
			return fmt.Errorf("complete result invariant failed")
		}
		return nil
	}
	if result.Outcome != "HARD_STOP" || result.Bundle != nil || result.HardStop == nil || result.EvidenceManifestSHA256 != nil {
		return fmt.Errorf("hard-stop result invariant failed")
	}
	return nil
}

func hmacDigest(key []byte, value string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func newUUIDv4() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic("crypto/rand unavailable")
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value[0:4], value[4:6], value[6:8], value[8:10], value[10:16])
}

func FetchFromEnvironment(ctx context.Context, args map[string]interface{}) (Result, error) {
	input, err := ParseInput(args)
	if err != nil {
		return Result{}, err
	}
	attemptID := newUUIDv4()
	service, err := loadEnvironmentService(attemptID)
	if err != nil {
		result := baseResult(input, attemptID)
		return hardStopResult(result, "configuration", "SEALED_CONFIG_UNAVAILABLE", "sealed WSDL fetch configuration is unavailable"), nil
	}
	return service.fetchInput(ctx, input), nil
}

func loadEnvironmentService(attemptID string) (*Service, error) {
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
	key, err := os.ReadFile(manifest.EvidenceHMACKeyFile)
	if err != nil || len(key) < 32 {
		return nil, fmt.Errorf("evidence key unavailable")
	}
	keyInfo, err := os.Lstat(manifest.EvidenceHMACKeyFile)
	if err != nil || !keyInfo.Mode().IsRegular() || keyInfo.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("evidence key unsafe")
	}
	binarySHA, err := executableSHA256()
	if err != nil {
		return nil, err
	}
	return NewService(ServiceConfig{
		Manifest:       manifest,
		ManifestSHA256: manifestSHA,
		Ledger:         &FilePermitLedger{Dir: manifest.PermitDir, BinarySHA256: binarySHA},
		Credentials:    credentials,
		EvidenceKey:    key,
		AttemptID:      func() string { return attemptID },
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

func publishEvidence(manifest Manifest, attemptID string, bundle *Bundle) (string, error) {
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
	committed = true
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}
