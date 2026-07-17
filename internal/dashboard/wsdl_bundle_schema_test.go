package dashboard

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/bridge"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/config"
)

func TestSAPWSDLBundleFetchToolPublishesExactSchemas(t *testing.T) {
	odataBridge, err := bridge.NewODataMCPBridge(&config.Config{})
	if err != nil {
		t.Fatalf("create bridge: %v", err)
	}
	NewHierarchicalRuntime(odataBridge, config.Config{})

	response := handleMCPRequest(t, odataBridge, "tools/list", nil)
	if response.Error != nil {
		t.Fatalf("tools/list returned error: %+v", response.Error)
	}
	var result struct {
		Tools []struct {
			Name         string                 `json:"name"`
			InputSchema  map[string]interface{} `json:"inputSchema"`
			OutputSchema map[string]interface{} `json:"outputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode tools/list: %v", err)
	}

	wantInput := readGoldenSchema(t, "testdata/sap_wsdl_bundle_fetch_once.input.schema.json")
	wantOutput := readGoldenSchema(t, "testdata/sap_wsdl_bundle_fetch_once.output.schema.json")
	for _, tool := range result.Tools {
		if tool.Name != "sap_wsdl_bundle_fetch_once" {
			continue
		}
		if !reflect.DeepEqual(tool.InputSchema, wantInput) {
			t.Fatalf("inputSchema mismatch\n got: %#v\nwant: %#v", tool.InputSchema, wantInput)
		}
		if !reflect.DeepEqual(tool.OutputSchema, wantOutput) {
			t.Fatalf("outputSchema mismatch\n got: %#v\nwant: %#v", tool.OutputSchema, wantOutput)
		}
		if got := schemaDigest(t, tool.InputSchema); got != "94dd1a4f23157cd0076a685b8104d2cddec090fec99b6fc1a624cbc334007ea2" {
			t.Fatalf("inputSchema canonical digest drifted: %s", got)
		}
		if got := schemaDigest(t, tool.OutputSchema); got != "391984c900fe9dcb3fa2560fd733ecd517edc3b77d190e40202c359eba590af4" {
			t.Fatalf("outputSchema canonical digest drifted: %s", got)
		}
		return
	}
	t.Fatal("tool sap_wsdl_bundle_fetch_once was not listed")
}

func TestSAPWSDLBundleFetchToolFailsClosedWithoutSealedConfiguration(t *testing.T) {
	t.Setenv("SAP_WSDL_BUNDLE_MANIFEST_FILE", "")
	odataBridge, err := bridge.NewODataMCPBridge(&config.Config{})
	if err != nil {
		t.Fatalf("create bridge: %v", err)
	}
	NewHierarchicalRuntime(odataBridge, config.Config{})
	response := handleMCPRequest(t, odataBridge, "tools/call", map[string]interface{}{
		"name": "sap_wsdl_bundle_fetch_once",
		"arguments": map[string]interface{}{
			"system_id":               "gpi_100",
			"contract_id":             "employee-shop-invoice-wsdl",
			"request_manifest_sha256": strings.Repeat("a", 64),
			"permit_id":               "6ba7b810-9dad-41d1-80b4-00c04fd430c8",
		},
	})
	if response.Error != nil {
		t.Fatalf("configuration hard stop should be a structured result: %+v", response.Error)
	}
	var result struct {
		StructuredContent struct {
			Outcome            string `json:"outcome"`
			PermitConsumed     bool   `json:"permit_consumed"`
			NetworkGetsStarted int    `json:"network_gets_started"`
			HardStop           struct {
				Code string `json:"code"`
			} `json:"hard_stop"`
		} `json:"structuredContent"`
	}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode tools/call: %v", err)
	}
	got := result.StructuredContent
	if got.Outcome != "HARD_STOP" || got.PermitConsumed || got.NetworkGetsStarted != 0 || got.HardStop.Code != "SEALED_CONFIG_UNAVAILABLE" {
		t.Fatalf("unexpected configuration hard stop: %#v", got)
	}
}

func TestSAPWSDLBundleFetchToolValidatesArgumentsBeforeConfigurationOrNetwork(t *testing.T) {
	t.Setenv("SAP_WSDL_BUNDLE_MANIFEST_FILE", "/path/that/must/not/be/read")
	odataBridge, err := bridge.NewODataMCPBridge(&config.Config{})
	if err != nil {
		t.Fatalf("create bridge: %v", err)
	}
	NewHierarchicalRuntime(odataBridge, config.Config{})
	response := handleMCPRequest(t, odataBridge, "tools/call", map[string]interface{}{
		"name": "sap_wsdl_bundle_fetch_once",
		"arguments": map[string]interface{}{
			"system_id":               "gpi_100",
			"contract_id":             "employee-shop-invoice-wsdl",
			"request_manifest_sha256": strings.Repeat("a", 64),
			"permit_id":               "6ba7b810-9dad-41d1-80b4-00c04fd430c8",
			"url":                     "https://caller-controlled.invalid/forbidden",
		},
	})
	if response.Error == nil {
		t.Fatal("additional caller-controlled property unexpectedly passed handler validation")
	}
	if !strings.Contains(strings.ToUpper(string(response.Error.Data)), "INVALID_INPUT") {
		t.Fatalf("unexpected validation error: %+v", response.Error)
	}
}

func readGoldenSchema(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden schema %s: %v", path, err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode golden schema %s: %v", path, err)
	}
	return schema
}

func schemaDigest(t *testing.T, schema map[string]interface{}) string {
	t.Helper()
	canonical, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("canonicalize schema: %v", err)
	}
	digest := sha256.Sum256(canonical)
	return fmt.Sprintf("%x", digest)
}
