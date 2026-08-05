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
	var cachedOutputSchema map[string]interface{}
	for _, tool := range listResult.Tools {
		if tool.Name != cachedCSDLSummaryToolName {
			continue
		}
		found = true
		cachedOutputSchema = tool.OutputSchema
		if tool.OutputSchema["type"] != "object" {
			t.Fatalf("missing object outputSchema: %#v", tool.OutputSchema)
		}
		properties, ok := tool.OutputSchema["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("outputSchema properties missing: %#v", tool.OutputSchema)
		}
		functionImports, ok := properties["function_imports"].(map[string]interface{})
		if !ok || functionImports["type"] != "array" {
			t.Fatalf("function_imports outputSchema missing: %#v", properties["function_imports"])
		}
		items, ok := functionImports["items"].(map[string]interface{})
		if !ok {
			t.Fatalf("function_imports item schema missing: %#v", functionImports)
		}
		if items["type"] != "object" || items["additionalProperties"] != false {
			t.Fatalf("function_imports item schema is not strict: %#v", items)
		}
		assertSchemaRequiredFields(t, items, []string{"name", "http_method", "return_type", "parameter_count", "is_action"})
		itemProperties, ok := items["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("function_imports item properties missing: %#v", items)
		}
		expectedTypes := map[string]string{
			"name":            "string",
			"http_method":     "string",
			"return_type":     "string",
			"parameter_count": "integer",
			"is_action":       "boolean",
		}
		for field, expectedType := range expectedTypes {
			fieldSchema, ok := itemProperties[field].(map[string]interface{})
			if !ok || fieldSchema["type"] != expectedType {
				t.Fatalf("function_imports item field %q has wrong schema: %#v", field, itemProperties[field])
			}
			if field == "parameter_count" && fieldSchema["minimum"] != float64(0) {
				t.Fatalf("parameter_count must have minimum 0: %#v", fieldSchema)
			}
		}
	}
	if !found {
		t.Fatalf("tool %q was not listed", cachedCSDLSummaryToolName)
	}
	if cachedOutputSchema["additionalProperties"] != false {
		t.Fatalf("cached summary output schema is not strict: %#v", cachedOutputSchema)
	}
	assertSchemaRequiredFields(t, cachedOutputSchema, []string{"digest_sha256", "version", "schema_namespace", "container_name", "entity_sets", "function_imports"})

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

func assertSchemaRequiredFields(t *testing.T, schema map[string]interface{}, expected []string) {
	t.Helper()
	values, ok := schema["required"].([]interface{})
	if !ok {
		t.Fatalf("schema required fields missing: %#v", schema)
	}
	actual := make(map[string]bool, len(values))
	for _, value := range values {
		name, ok := value.(string)
		if !ok {
			t.Fatalf("schema required field is not a string: %#v", value)
		}
		actual[name] = true
	}
	for _, name := range expected {
		if !actual[name] {
			t.Fatalf("schema required field %q missing: %#v", name, values)
		}
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
		FunctionImports: map[string]*models.FunctionImport{
			"z_action": {
				Name:       "ZAction",
				HTTPMethod: "POST",
				ReturnType: "Edm.Boolean",
				Parameters: []*models.FunctionParameter{
					{Name: "credential-like-parameter", Type: "Edm.String"},
				},
				IsAction: true,
			},
			"a_function": {
				Name:       "AFunction",
				HTTPMethod: "GET",
				ReturnType: "Edm.String",
				Parameters: []*models.FunctionParameter{
					{Name: "raw-secret-name", Type: "Edm.String"},
					{Name: "raw-secret-type", Type: "Edm.Int32"},
				},
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
	if len(result.FunctionImports) != 2 {
		t.Fatalf("function imports missing from cached summary: %#v", result.FunctionImports)
	}
	if result.FunctionImports[0].Name != "AFunction" || result.FunctionImports[1].Name != "ZAction" {
		t.Fatalf("function imports are not sorted by name: %#v", result.FunctionImports)
	}
	if result.FunctionImports[0].HTTPMethod != "GET" || result.FunctionImports[0].ReturnType != "Edm.String" || result.FunctionImports[0].ParameterCount != 2 || result.FunctionImports[0].IsAction {
		t.Fatalf("unexpected function contract: %#v", result.FunctionImports[0])
	}
	if result.FunctionImports[1].HTTPMethod != "POST" || result.FunctionImports[1].ReturnType != "Edm.Boolean" || result.FunctionImports[1].ParameterCount != 1 || !result.FunctionImports[1].IsAction {
		t.Fatalf("unexpected action contract: %#v", result.FunctionImports[1])
	}
	firstDigest := result.DigestSHA256
	runtime.mu.Lock()
	runtime.metadataCache["fixture"].FunctionImports = map[string]*models.FunctionImport{
		"a_function": {
			Name:       "AFunction",
			HTTPMethod: "GET",
			ReturnType: "Edm.String",
			Parameters: []*models.FunctionParameter{
				{Name: "raw-secret-name", Type: "Edm.String"},
				{Name: "raw-secret-type", Type: "Edm.Int32"},
			},
		},
		"z_action": {
			Name:       "ZAction",
			HTTPMethod: "POST",
			ReturnType: "Edm.Boolean",
			Parameters: []*models.FunctionParameter{
				{Name: "credential-like-parameter", Type: "Edm.String"},
			},
			IsAction: true,
		},
	}
	runtime.mu.Unlock()
	second := callCachedCSDLSummary(t, odataBridge)
	if second.DigestSHA256 != firstDigest {
		t.Fatalf("digest changed after equivalent function map reconstruction: first=%q second=%q", firstDigest, second.DigestSHA256)
	}
	runtime.mu.Lock()
	runtime.metadataCache["fixture"].FunctionImports["m_function"] = &models.FunctionImport{
		Name:       "MFunction",
		HTTPMethod: "GET",
		ReturnType: "Edm.Int64",
	}
	runtime.mu.Unlock()
	third := callCachedCSDLSummary(t, odataBridge)
	if third.DigestSHA256 == firstDigest {
		t.Fatalf("digest did not change after function import contract change: digest=%q", third.DigestSHA256)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("encode summary: %v", err)
	}
	for _, forbidden := range []string{"credential-like-parameter", "raw-secret-name", "raw-secret-type", "Edm.Int32"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("summary leaked forbidden function metadata %q: %s", forbidden, encoded)
		}
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
	if first.FunctionImports == nil {
		t.Fatal("backward-compatible empty function_imports must be an array")
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
	FunctionImports []decodedCachedCSDLFunction  `json:"function_imports"`
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

type decodedCachedCSDLFunction struct {
	Name           string `json:"name"`
	HTTPMethod     string `json:"http_method"`
	ReturnType     string `json:"return_type"`
	ParameterCount int    `json:"parameter_count"`
	IsAction       bool   `json:"is_action"`
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
