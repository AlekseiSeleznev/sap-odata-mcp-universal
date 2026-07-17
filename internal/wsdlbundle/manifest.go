package wsdlbundle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"reflect"
	"strings"
)

func ManifestSHA256(manifest Manifest) (string, error) {
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("encode manifest: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func validateManifest(manifest Manifest, production bool) error {
	if manifest.SchemaVersion != 1 || manifest.SystemID != SystemID || manifest.ContractID != ContractID {
		return fmt.Errorf("MANIFEST_IDENTITY_MISMATCH")
	}
	root, err := url.Parse(manifest.RootURL)
	if err != nil || root.Host == "" || root.User != nil || root.Fragment != "" {
		return fmt.Errorf("MANIFEST_TARGET_INVALID")
	}
	if root.Scheme != "https" && !(root.Scheme == "http" && isLoopbackHost(root.Hostname())) {
		return fmt.Errorf("MANIFEST_TARGET_INVALID")
	}
	if canonicalOrigin(root) != manifest.AllowedOrigin {
		return fmt.Errorf("MANIFEST_ORIGIN_MISMATCH")
	}
	query := root.Query()
	clients, ok := query["sap-client"]
	if !ok || len(clients) != 1 || clients[0] != manifest.SAPClient || manifest.SAPClient == "" {
		return fmt.Errorf("MANIFEST_CLIENT_MISMATCH")
	}
	if manifest.ExpectedServiceQName == "" || manifest.ExpectedPortQName == "" || manifest.ExpectedBindingQName == "" || manifest.ExpectedOperation == "" || manifest.ExpectedSOAPAction == "" || manifest.EvidenceDir == "" {
		return fmt.Errorf("MANIFEST_CONTRACT_INCOMPLETE")
	}
	if err := validateLimits(manifest.Limits); err != nil {
		return err
	}
	if production && !reflect.DeepEqual(manifest.Limits, ProductionLimits()) {
		return fmt.Errorf("MANIFEST_LIMITS_MISMATCH")
	}
	return nil
}

func validateLimits(l Limits) error {
	if l.ConnectTimeoutMS <= 0 || l.TLSHandshakeTimeoutMS <= 0 || l.ResponseHeaderTimeoutMS <= 0 || l.PerDocumentTimeoutMS <= 0 || l.WholeActionTimeoutMS <= 0 ||
		l.MaxDepth <= 0 || l.MaxDocuments <= 0 || l.MaxDocuments > 64 || l.MaxReferences <= 0 || l.MaxReferences > 256 ||
		l.MaxDocumentBytes <= 0 || l.MaxTotalBytes < l.MaxDocumentBytes || l.MaxXMLTokens <= 0 || l.MaxXMLNesting <= 0 || l.MaxAttributes <= 0 || l.MaxAttributeBytes <= 0 || l.MaxEvidenceBytes <= 0 {
		return fmt.Errorf("MANIFEST_LIMITS_INVALID")
	}
	return nil
}

func canonicalOrigin(parsed *url.URL) string {
	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if (parsed.Scheme == "https" && port == "443") || (parsed.Scheme == "http" && port == "80") {
		port = ""
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port != "" {
		host += ":" + port
	}
	return strings.ToLower(parsed.Scheme) + "://" + host
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func readPrivateJSON(path string, target interface{}, max int64) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("PRIVATE_FILE_UNAVAILABLE")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("PRIVATE_FILE_UNAVAILABLE")
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, max))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("PRIVATE_FILE_INVALID")
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("PRIVATE_FILE_INVALID")
	}
	return nil
}
