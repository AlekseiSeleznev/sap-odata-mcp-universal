package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/bridge"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/config"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/models"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/transport"
)

const cachedCSDLSummaryToolName = "odata_cached_csdl_summary"

func TestCachedCSDLSummaryToolIsDocumentedAndFailsClosedOnCacheMiss(t *testing.T) {
	odataBridge, err := bridge.NewODataMCPBridge(&config.Config{})
	if err != nil {
		t.Fatalf("create bridge: %v", err)
	}
	NewHierarchicalRuntime(odataBridge, config.Config{})

	initializeResponse := handleMCPRequest(t, odataBridge, "initialize", map[string]interface{}{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "offline-fixture", "version": "test"},
	})
	if initializeResponse.Error != nil {
		t.Fatalf("initialize returned error: %+v", initializeResponse.Error)
	}

	listResponse := handleMCPRequest(t, odataBridge, "tools/list", nil)
	if listResponse.Error != nil {
		t.Fatalf("tools/list returned error: %+v", listResponse.Error)
	}

	var listResult struct {
		Tools []struct {
			Name         string                 `json:"name"`
			OutputSchema map[string]interface{} `json:"outputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(listResponse.Result, &listResult); err != nil {
		t.Fatalf("decode tools/list: %v", err)
	}

	var found bool
	for _, tool := range listResult.Tools {
		if tool.Name != cachedCSDLSummaryToolName {
			continue
		}
		found = true
		if tool.OutputSchema["type"] != "object" {
			t.Fatalf("missing object outputSchema: %#v", tool.OutputSchema)
		}
	}
	if !found {
		t.Fatalf("tool %q was not listed", cachedCSDLSummaryToolName)
	}

	callResponse := handleMCPRequest(t, odataBridge, "tools/call", map[string]interface{}{
		"name": cachedCSDLSummaryToolName,
		"arguments": map[string]interface{}{
			"system_id":  "gpi-100",
			"service_id": "sales-order",
		},
	})
	if callResponse.Error == nil {
		t.Fatal("cache miss unexpectedly succeeded")
	}
	if !strings.Contains(strings.ToLower(string(callResponse.Error.Data)), "cached metadata unavailable") {
		t.Fatalf("unexpected cache miss error: %+v", callResponse.Error)
	}
}

func TestCachedCSDLSummaryHonorsRequestCancellationWhileRuntimeIsBusy(t *testing.T) {
	odataBridge, err := bridge.NewODataMCPBridge(&config.Config{})
	if err != nil {
		t.Fatalf("create bridge: %v", err)
	}
	runtime := NewHierarchicalRuntime(odataBridge, config.Config{})

	locked := make(chan struct{})
	unlock := make(chan struct{})
	go func() {
		runtime.mu.Lock()
		close(locked)
		<-unlock
		runtime.mu.Unlock()
	}()
	<-locked
	defer close(unlock)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	paramsJSON, err := json.Marshal(map[string]interface{}{
		"name": cachedCSDLSummaryToolName,
		"arguments": map[string]interface{}{
			"system_id":  "gpi-100",
			"service_id": "sales-order",
		},
	})
	if err != nil {
		t.Fatalf("encode params: %v", err)
	}

	response, err := odataBridge.GetServer().HandleMessage(ctx, &transport.Message{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "tools/call",
		Params:  paramsJSON,
	})
	if err != nil {
		t.Fatalf("handle tools/call: %v", err)
	}
	if response.Error == nil || !strings.Contains(strings.ToLower(string(response.Error.Data)), "deadline exceeded") {
		t.Fatalf("expected bounded cancellation error, got %+v", response)
	}
}

func TestCachedCSDLSummaryPublicSeamUsesInMemoryFixture(t *testing.T) {
	odataBridge, err := bridge.NewODataMCPBridge(&config.Config{})
	if err != nil {
		t.Fatalf("create bridge: %v", err)
	}
	runtime := NewHierarchicalRuntime(odataBridge, config.Config{})
	runtime.mu.Lock()
	runtime.serviceCacheKeys[serviceBindingKey("gpi-100", "sales-order")] = "fixture"
	runtime.metadataCache["fixture"] = &models.ODataMetadata{
		Version:         "1.0",
		SchemaNamespace: "Fixture.Service",
		ContainerName:   "FixtureContainer",
		EntitySets: map[string]*models.EntitySet{
			"SalesOrderSet": {Name: "SalesOrderSet", EntityType: "Fixture.Service.SalesOrder"},
		},
		EntityTypes: map[string]*models.EntityType{
			"Fixture.Service.SalesOrder": {
				Name:          "SalesOrder",
				KeyProperties: []string{"ID"},
				Properties:    []*models.EntityProperty{{Name: "ID", Type: "Edm.String", IsKey: true, Nullable: false}},
			},
		},
	}
	runtime.mu.Unlock()

	if response := handleMCPRequest(t, odataBridge, "initialize", map[string]interface{}{}); response.Error != nil {
		t.Fatalf("initialize returned error: %+v", response.Error)
	}
	listResponse := handleMCPRequest(t, odataBridge, "tools/list", nil)
	if listResponse.Error != nil || !strings.Contains(string(listResponse.Result), cachedCSDLSummaryToolName) {
		t.Fatalf("tools/list did not expose cached summary: %+v", listResponse)
	}
	result := callCachedCSDLSummary(t, odataBridge)
	if len(result.EntitySets) != 1 || result.EntitySets[0].Name != "SalesOrderSet" {
		t.Fatalf("unexpected in-memory fixture result: %#v", result)
	}
}

func TestCachedCSDLSummaryReturnsSanitizedDeterministicFacetsWithoutNetwork(t *testing.T) {
	var requests atomic.Int32
	metadataXML := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
  <edmx:DataServices xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="ZMCP_SO_SALES_ORDER_1C_RU_SRV" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="DeliveryItem">
        <Key>
          <PropertyRef Name="Vbeln"/>
          <PropertyRef Name="Posnr"/>
        </Key>
        <Property Name="Vbeln" Type="Edm.String" Nullable="false" MaxLength="10"/>
        <Property Name="Posnr" Type="Edm.String" Nullable="false" MaxLength="6"/>
        <Property Name="Uecha" Type="Edm.String" Nullable="false" MaxLength="6"/>
        <Property Name="Quantity" Type="Edm.Decimal" Nullable="true" Precision="13" Scale="3"/>
      </EntityType>
      <EntityContainer Name="ZMCP_SO_SALES_ORDER_1C_RU_SRV_Entities" m:IsDefaultEntityContainer="true">
        <EntitySet Name="DeliveryItemSet" EntityType="ZMCP_SO_SALES_ORDER_1C_RU_SRV.DeliveryItem"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests.Add(1)
		if !strings.HasSuffix(req.URL.Path, "/$metadata") {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(metadataXML))
	}))
	defer server.Close()

	odataBridge, err := bridge.NewODataMCPBridge(&config.Config{})
	if err != nil {
		t.Fatalf("create bridge: %v", err)
	}
	runtime := NewHierarchicalRuntime(odataBridge, config.Config{})
	system := SystemInfo{
		ID:       "gpi-100",
		Username: "runtime-user",
		Services: []ServiceInfo{{
			ID:         "sales-order",
			Name:       "ZMCP_SO_SALES_ORDER_1C_RU_SRV",
			ServiceURL: server.URL + "/sap/opu/odata/sap/ZMCP_SO_SALES_ORDER_1C_RU_SRV",
		}},
		Entities: []EntityInfo{{
			ID: "delivery-items",
			Operations: []OperationInfo{{
				ID:        "delivery-items-list",
				Verb:      "list",
				ServiceID: "sales-order",
				EntitySet: "DeliveryItemSet",
				Enabled:   true,
			}},
		}},
	}
	if err := runtime.ApplySystem(context.Background(), system); err != nil {
		t.Fatalf("prime runtime metadata cache: %v", err)
	}
	requestsAfterActivation := requests.Load()
	if requestsAfterActivation != 1 {
		t.Fatalf("expected one metadata request, got %d", requestsAfterActivation)
	}

	first := callCachedCSDLSummary(t, odataBridge)
	second := callCachedCSDLSummary(t, odataBridge)
	if requests.Load() != requestsAfterActivation {
		t.Fatalf("summary performed a network request: before=%d after=%d", requestsAfterActivation, requests.Load())
	}
	if first.DigestSHA256 == "" || first.DigestSHA256 != second.DigestSHA256 {
		t.Fatalf("digest is empty or unstable: first=%q second=%q", first.DigestSHA256, second.DigestSHA256)
	}
	if first.Version != "1.0" || first.SchemaNamespace != "ZMCP_SO_SALES_ORDER_1C_RU_SRV" {
		t.Fatalf("unexpected metadata identity: %#v", first)
	}
	if len(first.EntitySets) != 1 {
		t.Fatalf("unexpected entity sets: %#v", first.EntitySets)
	}
	entitySet := first.EntitySets[0]
	if entitySet.Name != "DeliveryItemSet" || entitySet.EntityType != "ZMCP_SO_SALES_ORDER_1C_RU_SRV.DeliveryItem" {
		t.Fatalf("unexpected entity set mapping: %#v", entitySet)
	}
	if strings.Join(entitySet.Keys, ",") != "Posnr,Vbeln" {
		t.Fatalf("keys are missing or not canonical: %#v", entitySet.Keys)
	}
	uecha := findSummaryProperty(t, entitySet.Properties, "Uecha")
	if uecha.Type != "Edm.String" || uecha.Nullable || uecha.MaxLength != "6" {
		t.Fatalf("unexpected Uecha contract: %#v", uecha)
	}
	quantity := findSummaryProperty(t, entitySet.Properties, "Quantity")
	if quantity.Precision != "13" || quantity.Scale != "3" {
		t.Fatalf("decimal facets were not preserved: %#v", quantity)
	}

	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("encode summary: %v", err)
	}
	for _, forbidden := range []string{server.URL, "runtime-user", "service_root", "parsed_at"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("summary leaked forbidden value %q: %s", forbidden, encoded)
		}
	}
}

type decodedCachedCSDLSummary struct {
	DigestSHA256    string                       `json:"digest_sha256"`
	Version         string                       `json:"version"`
	SchemaNamespace string                       `json:"schema_namespace"`
	ContainerName   string                       `json:"container_name"`
	EntitySets      []decodedCachedCSDLEntitySet `json:"entity_sets"`
}

type decodedCachedCSDLEntitySet struct {
	Name       string                      `json:"name"`
	EntityType string                      `json:"entity_type"`
	Keys       []string                    `json:"keys"`
	Properties []decodedCachedCSDLProperty `json:"properties"`
}

type decodedCachedCSDLProperty struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Nullable  bool   `json:"nullable"`
	IsKey     bool   `json:"is_key"`
	MaxLength string `json:"max_length"`
	Precision string `json:"precision"`
	Scale     string `json:"scale"`
}

func callCachedCSDLSummary(t *testing.T, odataBridge *bridge.ODataMCPBridge) decodedCachedCSDLSummary {
	t.Helper()
	response := handleMCPRequest(t, odataBridge, "tools/call", map[string]interface{}{
		"name": cachedCSDLSummaryToolName,
		"arguments": map[string]interface{}{
			"system_id":  "gpi-100",
			"service_id": "sales-order",
		},
	})
	if response.Error != nil {
		t.Fatalf("cached summary returned error: %+v", response.Error)
	}
	var result struct {
		StructuredContent decodedCachedCSDLSummary `json:"structuredContent"`
	}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode cached summary: %v", err)
	}
	return result.StructuredContent
}

func findSummaryProperty(t *testing.T, properties []decodedCachedCSDLProperty, name string) decodedCachedCSDLProperty {
	t.Helper()
	for _, property := range properties {
		if property.Name == name {
			return property
		}
	}
	t.Fatalf("property %q not found: %#v", name, properties)
	return decodedCachedCSDLProperty{}
}

func handleMCPRequest(t *testing.T, odataBridge *bridge.ODataMCPBridge, method string, params interface{}) *transport.Message {
	t.Helper()
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("encode params: %v", err)
	}
	response, err := odataBridge.GetServer().HandleMessage(context.Background(), &transport.Message{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  method,
		Params:  paramsJSON,
	})
	if err != nil {
		t.Fatalf("handle %s: %v", method, err)
	}
	return response
}
