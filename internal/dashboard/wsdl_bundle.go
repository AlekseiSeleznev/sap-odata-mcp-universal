package dashboard

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/mcp"
	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/wsdlbundle"
)

const wsdlBundleFetchToolName = "sap_wsdl_bundle_fetch_once"

var (
	//go:embed testdata/sap_wsdl_bundle_fetch_once.input.schema.json
	wsdlBundleInputSchemaJSON []byte
	//go:embed testdata/sap_wsdl_bundle_fetch_once.output.schema.json
	wsdlBundleOutputSchemaJSON []byte

	wsdlBundleSchemasOnce  sync.Once
	wsdlBundleInputSchema  map[string]interface{}
	wsdlBundleOutputSchema map[string]interface{}
	wsdlBundleSchemasErr   error
)

func (r *HierarchicalRuntime) registerWSDLBundleFetchTool() {
	inputSchema, outputSchema, err := loadWSDLBundleSchemas()
	if err != nil {
		panic(fmt.Sprintf("load %s schemas: %v", wsdlBundleFetchToolName, err))
	}
	r.bridge.GetServer().AddTool(&mcp.Tool{
		Name:         wsdlBundleFetchToolName,
		Description:  "Fetch the sealed GPI Employee Shop Invoice WSDL 1.1/XSD 1.0/WS-Policy closure exactly once under an atomically consumed permit. Returns sanitized atomic evidence only; no redirect, retry, fallback, parameter override, activation, or SOAP POST.",
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
	}, r.wsdlBundleFetchHandler)
}

func loadWSDLBundleSchemas() (map[string]interface{}, map[string]interface{}, error) {
	wsdlBundleSchemasOnce.Do(func() {
		if err := json.Unmarshal(wsdlBundleInputSchemaJSON, &wsdlBundleInputSchema); err != nil {
			wsdlBundleSchemasErr = fmt.Errorf("decode input schema: %w", err)
			return
		}
		if err := json.Unmarshal(wsdlBundleOutputSchemaJSON, &wsdlBundleOutputSchema); err != nil {
			wsdlBundleSchemasErr = fmt.Errorf("decode output schema: %w", err)
		}
	})
	return wsdlBundleInputSchema, wsdlBundleOutputSchema, wsdlBundleSchemasErr
}

func (r *HierarchicalRuntime) wsdlBundleFetchHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	result, err := wsdlbundle.FetchFromEnvironment(ctx, args, r.activeSystemID())
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *HierarchicalRuntime) activeSystemID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.activeSystem
}
