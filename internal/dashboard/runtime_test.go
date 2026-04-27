package dashboard

import "testing"

func TestMakeToolNameUsesOperationIDForSameVerbBindings(t *testing.T) {
	system := SystemInfo{ID: "gpd-100", Name: "GPD"}
	entity := EntityInfo{ID: "materials", Label: "Materials"}
	used := map[string]struct{}{}

	first := makeToolName(system, entity, OperationInfo{
		ID:        "materials-get-materialset",
		Name:      "GET MaterialSet",
		Verb:      "get",
		EntitySet: "MaterialSet",
	}, used)
	second := makeToolName(system, entity, OperationInfo{
		ID:        "materials-get-descriptionset",
		Name:      "GET DescriptionSet",
		Verb:      "get",
		EntitySet: "DescriptionSet",
	}, used)

	if first != "materials_get_materialset_for_gpd_100" {
		t.Fatalf("unexpected first tool name: %s", first)
	}
	if second != "materials_get_descriptionset_for_gpd_100" {
		t.Fatalf("unexpected second tool name: %s", second)
	}
}

func TestMakeToolNameKeepsLegacySingleVerbName(t *testing.T) {
	got := makeToolName(
		SystemInfo{ID: "gpd-100", Name: "GPD"},
		EntityInfo{ID: "materials", Label: "Materials"},
		OperationInfo{ID: "materials-get", Verb: "get", EntitySet: "MaterialSet"},
		map[string]struct{}{},
	)
	if got != "materials_get_for_gpd_100" {
		t.Fatalf("unexpected legacy tool name: %s", got)
	}
}

func TestOperationQueryOptionsMergeDefaultsAndCallArgs(t *testing.T) {
	got := operationQueryOptions(
		OperationInfo{
			Query: map[string]string{
				"$expand":  "ToDescription",
				"$orderby": "MATNR",
			},
		},
		map[string]interface{}{
			"$top":    float64(1),
			"$expand": "ToPlant",
		},
	)

	if got["$expand"] != "ToPlant" {
		t.Fatalf("call arg did not override default expand: %#v", got)
	}
	if got["$orderby"] != "MATNR" {
		t.Fatalf("default orderby was not preserved: %#v", got)
	}
	if got["$top"] != "1" {
		t.Fatalf("top arg was not normalized: %#v", got)
	}
}

func TestSameOperationBindingIncludesDefaultQuery(t *testing.T) {
	plain := OperationInfo{Verb: "list", ServiceID: "zmat-data-srv", EntitySet: "MaterialSet"}
	expanded := OperationInfo{
		Verb:      "list",
		ServiceID: "zmat-data-srv",
		EntitySet: "MaterialSet",
		Query:     map[string]string{"expand": "ToDescription"},
	}
	expandedDuplicate := OperationInfo{
		Verb:      "list",
		ServiceID: "zmat-data-srv",
		EntitySet: "MaterialSet",
		Query:     map[string]string{"$expand": "ToDescription"},
	}

	if sameOperationBinding(plain, expanded) {
		t.Fatal("plain and expanded bindings should be distinct")
	}
	if !sameOperationBinding(expanded, expandedDuplicate) {
		t.Fatal("normalized query aliases should compare as the same binding")
	}
}
