package wsdlbundle

import "testing"

func TestParseInputAcceptsOnlyTheSealedContractShape(t *testing.T) {
	valid := map[string]interface{}{
		"system_id":               "gpi_100",
		"contract_id":             "employee-shop-invoice-wsdl",
		"request_manifest_sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"permit_id":               "6ba7b810-9dad-41d1-80b4-00c04fd430c8",
	}
	got, err := ParseInput(valid)
	if err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
	if got.SystemID != "gpi_100" || got.ContractID != "employee-shop-invoice-wsdl" {
		t.Fatalf("unexpected parsed input: %#v", got)
	}

	tests := []struct {
		name   string
		mutate func(map[string]interface{})
	}{
		{"missing field", func(v map[string]interface{}) { delete(v, "permit_id") }},
		{"additional property", func(v map[string]interface{}) { v["url"] = "https://forbidden.invalid" }},
		{"wrong system", func(v map[string]interface{}) { v["system_id"] = "gpd_100" }},
		{"wrong contract", func(v map[string]interface{}) { v["contract_id"] = "other" }},
		{"uppercase digest", func(v map[string]interface{}) {
			v["request_manifest_sha256"] = "A123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		}},
		{"non v4 permit", func(v map[string]interface{}) { v["permit_id"] = "6ba7b810-9dad-11d1-80b4-00c04fd430c8" }},
		{"non string", func(v map[string]interface{}) { v["system_id"] = 100 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			candidate := make(map[string]interface{}, len(valid))
			for key, value := range valid {
				candidate[key] = value
			}
			tc.mutate(candidate)
			if _, err := ParseInput(candidate); err == nil {
				t.Fatal("invalid input unexpectedly passed validation")
			}
		})
	}
}
