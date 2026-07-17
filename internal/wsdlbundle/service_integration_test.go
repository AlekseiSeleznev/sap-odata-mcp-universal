package wsdlbundle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestServiceFetchesRecursiveClosureOnceAndPublishesSanitizedAtomicEvidence(t *testing.T) {
	const (
		username   = "fixture-user"
		password   = "fixture-password"
		soapAction = "urn:private:employee-shop:invoice"
	)
	counts := map[string]int{}
	var countsMu sync.Mutex
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		countsMu.Lock()
		counts[req.URL.Path]++
		countsMu.Unlock()
		gotUser, gotPassword, ok := req.BasicAuth()
		if !ok || gotUser != username || gotPassword != password {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if req.Header.Get("Accept-Encoding") != "identity" {
			http.Error(w, "encoding must be identity", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		switch req.URL.Path {
		case "/invoice.wsdl":
			fmt.Fprintf(w, `<?xml version="1.0"?>
<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" xmlns:tns="urn:employee-shop:invoice" xmlns:contract="urn:employee-shop:contract" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/" xmlns:wsp="http://www.w3.org/ns/ws-policy" xmlns:wsu="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" targetNamespace="urn:employee-shop:invoice">
  <wsdl:types><xsd:schema targetNamespace="urn:employee-shop:inline"><xsd:import namespace="urn:employee-shop:types" schemaLocation="/common.xsd"/></xsd:schema></wsdl:types>
  <wsdl:import namespace="urn:employee-shop:contract" location="/extra.wsdl"/>
  <wsp:Policy wsu:Id="local-policy"><wsp:ExactlyOne/></wsp:Policy>
  <wsp:PolicyReference URI="#local-policy"/>
  <wsp:PolicyReference URI="/policy.xml#transport-policy"/>
  <wsdl:service name="InvoiceService" wsp:PolicyURIs="/policy.xml#transport-policy"><wsdl:port name="InvoicePort" binding="contract:InvoiceBinding"><soap:address location="%s/private-soap-endpoint"/></wsdl:port></wsdl:service>
</wsdl:definitions>`, server.URL)
		case "/extra.wsdl":
			fmt.Fprintf(w, `<?xml version="1.0"?><wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" xmlns:tns="urn:employee-shop:contract" xmlns:types="urn:employee-shop:types" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/" targetNamespace="urn:employee-shop:contract"><wsdl:types><xsd:schema targetNamespace="urn:employee-shop:contract"><xsd:import namespace="urn:employee-shop:types" schemaLocation="/%%63ommon.xsd"/></xsd:schema></wsdl:types><wsdl:message name="InvoiceRequest"><wsdl:part name="parameters" element="types:InvoiceRequest"/></wsdl:message><wsdl:message name="InvoiceResponse"><wsdl:part name="parameters" element="types:InvoiceResponse"/></wsdl:message><wsdl:message name="InvoiceFault"><wsdl:part name="fault" element="types:InvoiceFault"/></wsdl:message><wsdl:portType name="InvoicePortType"><wsdl:operation name="SubmitInvoice"><wsdl:input message="tns:InvoiceRequest"/><wsdl:output message="tns:InvoiceResponse"/><wsdl:fault name="InvoiceFault" message="tns:InvoiceFault"/></wsdl:operation></wsdl:portType><wsdl:binding name="InvoiceBinding" type="tns:InvoicePortType"><soap:binding transport="http://schemas.xmlsoap.org/soap/http" style="document"/><wsdl:operation name="SubmitInvoice"><soap:operation soapAction="%s"/></wsdl:operation></wsdl:binding></wsdl:definitions>`, soapAction)
		case "/common.xsd":
			fmt.Fprint(w, `<?xml version="1.0"?><xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:types="urn:employee-shop:types" targetNamespace="urn:employee-shop:types"><xsd:include schemaLocation="/facets.xsd"/><xsd:element name="InvoiceRequest" type="types:InvoiceType"/><xsd:element name="InvoiceResponse" type="types:InvoiceType"/><xsd:element name="InvoiceFault" type="xsd:string"/><xsd:complexType name="InvoiceType"><xsd:sequence><xsd:element name="Currency" type="xsd:string" minOccurs="1" maxOccurs="1"/><xsd:element name="Amount" type="xsd:decimal" minOccurs="1" maxOccurs="1"/></xsd:sequence></xsd:complexType></xsd:schema>`)
		case "/facets.xsd":
			fmt.Fprint(w, `<?xml version="1.0"?><xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema"><xsd:redefine schemaLocation="/common.xsd"/><xsd:simpleType name="CurrencyCode"><xsd:restriction base="xsd:string"><xsd:length value="3"/><xsd:pattern value="[A-Z]{3}"/><xsd:enumeration value="USD"/><xsd:enumeration value="EUR"/></xsd:restriction></xsd:simpleType></xsd:schema>`)
		case "/policy.xml":
			fmt.Fprint(w, `<?xml version="1.0"?><wsp:Policy xmlns:wsp="http://www.w3.org/ns/ws-policy" xmlns:wsu="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" xmlns:sp="http://docs.oasis-open.org/ws-sx/ws-securitypolicy/200702" wsu:Id="transport-policy"><sp:TransportBinding/></wsp:Policy>`)
		default:
			http.NotFound(w, req)
		}
	}))
	defer server.Close()

	temp := t.TempDir()
	evidenceDir := filepath.Join(temp, "evidence")
	manifest := productionFixtureManifest(server.URL+"/invoice.wsdl?sap-client=100", evidenceDir)
	manifestSHA, err := ManifestSHA256(manifest)
	if err != nil {
		t.Fatalf("manifest digest: %v", err)
	}
	binarySHA := strings.Repeat("ab", 32)
	input := Input{SystemID: SystemID, ContractID: ContractID, RequestManifestSHA256: manifestSHA, PermitID: "6ba7b810-9dad-41d1-80b4-00c04fd430c8"}
	writeFixturePermit(t, temp, input, binarySHA)

	service, err := NewService(ServiceConfig{
		ActiveSystemID: SystemID,
		Manifest:       manifest,
		ManifestSHA256: manifestSHA,
		Ledger:         &FilePermitLedger{Dir: temp, BinarySHA256: binarySHA},
		Credentials:    Credentials{Username: username, Password: password},
		EvidenceKey:    []byte("0123456789abcdef0123456789abcdef"),
	})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	result, err := service.Fetch(context.Background(), map[string]interface{}{
		"system_id": input.SystemID, "contract_id": input.ContractID,
		"request_manifest_sha256": input.RequestManifestSHA256, "permit_id": input.PermitID,
	})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if result.Outcome != "COMPLETE" || !result.PermitConsumed || result.Bundle == nil || result.HardStop != nil {
		t.Fatalf("unexpected result: %#v hard_stop=%+v", result, result.HardStop)
	}
	if result.NetworkGetsStarted != 5 {
		t.Fatalf("network gets = %d, want five unique documents", result.NetworkGetsStarted)
	}
	countsMu.Lock()
	for _, path := range []string{"/invoice.wsdl", "/extra.wsdl", "/common.xsd", "/facets.xsd", "/policy.xml"} {
		if counts[path] != 1 {
			t.Fatalf("%s fetched %d times", path, counts[path])
		}
	}
	requestsBeforeReplay := totalCounts(counts)
	countsMu.Unlock()

	contract := result.Bundle.Contract
	if contract.ServiceQName != "{urn:employee-shop:invoice}InvoiceService" || contract.PortQName != "{urn:employee-shop:invoice}InvoicePort" || contract.BindingQName != "{urn:employee-shop:contract}InvoiceBinding" {
		t.Fatalf("service/port/binding proof missing: %#v", contract)
	}
	if contract.Operation != "SubmitInvoice" || contract.InputMessageQName != "{urn:employee-shop:contract}InvoiceRequest" || contract.OutputMessageQName == nil || *contract.OutputMessageQName != "{urn:employee-shop:contract}InvoiceResponse" || len(contract.FaultMessageQNames) != 1 || len(contract.Messages) != 3 {
		t.Fatalf("operation message proof missing: %#v", contract)
	}
	if !contract.SOAPActionMatchesSealedExpected || contract.SOAPActionSHA256 == "" || len(contract.PolicyAssertionQNames) != 1 {
		t.Fatalf("SOAP action or policy proof missing: %#v", contract)
	}
	if !hasFacet(contract.XSDComponents, "CurrencyCode", "length", "3") || !hasFacet(contract.XSDComponents, "CurrencyCode", "pattern", "[A-Z]{3}") {
		t.Fatalf("XSD facets missing: %#v", contract.XSDComponents)
	}
	if !hasFacet(contract.XSDComponents, "CurrencyCode", "enumeration", "USD") || !hasFacet(contract.XSDComponents, "CurrencyCode", "enumeration", "EUR") {
		t.Fatalf("repeated XSD facets missing: %#v", contract.XSDComponents)
	}
	if !hasOrderedElement(contract.XSDComponents, "{urn:employee-shop:types}InvoiceType", "Currency", 1) || !hasOrderedElement(contract.XSDComponents, "{urn:employee-shop:types}InvoiceType", "Amount", 2) {
		t.Fatalf("XSD sequence order missing: %#v", contract.XSDComponents)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("encode result: %v", err)
	}
	for _, forbidden := range []string{server.URL, username, password, soapAction, "sap-client=100"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("public result leaked forbidden value %q", forbidden)
		}
	}
	entries, err := os.ReadDir(evidenceDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("atomic evidence not published exactly once: entries=%d err=%v", len(entries), err)
	}
	info, err := entries[0].Info()
	if err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("evidence permissions are unsafe: info=%v err=%v", info, err)
	}

	replay, err := service.Fetch(context.Background(), map[string]interface{}{
		"system_id": input.SystemID, "contract_id": input.ContractID,
		"request_manifest_sha256": input.RequestManifestSHA256, "permit_id": input.PermitID,
	})
	if err != nil {
		t.Fatalf("replay should return a hard-stop envelope: %v", err)
	}
	if replay.Outcome != "HARD_STOP" || replay.PermitConsumed || replay.NetworkGetsStarted != 0 || replay.Bundle != nil || replay.HardStop == nil || replay.HardStop.Code != "PERMIT_REPLAY" {
		t.Fatalf("unexpected replay result: %#v", replay)
	}
	countsMu.Lock()
	defer countsMu.Unlock()
	if totalCounts(counts) != requestsBeforeReplay {
		t.Fatal("permit replay performed network I/O")
	}
}

func productionFixtureManifest(rootURL, evidenceDir string) Manifest {
	return Manifest{
		SchemaVersion: 1, SystemID: SystemID, ContractID: ContractID,
		RootURL: rootURL, SAPClient: "100", AllowedOrigin: originOf(rootURL),
		ExpectedServiceQName: "{urn:employee-shop:invoice}InvoiceService",
		ExpectedPortQName:    "{urn:employee-shop:invoice}InvoicePort",
		ExpectedBindingQName: "{urn:employee-shop:contract}InvoiceBinding",
		ExpectedOperation:    "SubmitInvoice",
		ExpectedSOAPAction:   "urn:private:employee-shop:invoice",
		EvidenceDir:          evidenceDir,
		Limits:               ProductionLimits(),
	}
}

func writeFixturePermit(t *testing.T, dir string, input Input, binarySHA string) {
	t.Helper()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod permit ledger: %v", err)
	}
	now := time.Now().UTC()
	permit := Permit{SchemaVersion: 1, PermitID: input.PermitID, Purpose: permitPurpose, SystemID: input.SystemID, ContractID: input.ContractID, RequestManifestSHA256: input.RequestManifestSHA256, BinarySHA256: binarySHA, NotBefore: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute)}
	body, err := json.Marshal(permit)
	if err != nil {
		t.Fatalf("encode permit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, input.PermitID+".json"), body, 0o600); err != nil {
		t.Fatalf("write permit: %v", err)
	}
}

func originOf(raw string) string {
	idx := strings.Index(raw, "/invoice.wsdl")
	if idx < 0 {
		return raw
	}
	return raw[:idx]
}

func totalCounts(counts map[string]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}

func hasFacet(components []XSDComponent, name, facet, value string) bool {
	for _, component := range components {
		if component.Name == name {
			for _, candidate := range component.Facets {
				if candidate.Name == facet && candidate.Value == value {
					return true
				}
			}
		}
	}
	return false
}

func hasOrderedElement(components []XSDComponent, parent, name string, order int) bool {
	for _, component := range components {
		if component.ParentQName == parent && component.Name == name && component.Order == order {
			return true
		}
	}
	return false
}
