package dashboard

import "testing"

func TestCurrentAccessModeFailsClosedWithoutMatchingActiveSystem(t *testing.T) {
	runtime := &HierarchicalRuntime{}
	if got := runtime.currentAccessMode("gpi_100"); got != "restricted" {
		t.Fatalf("empty runtime access mode = %q, want restricted", got)
	}

	runtime.activeSystem = "gpi_100"
	runtime.activeAccess = "unrestricted"
	if got := runtime.currentAccessMode("gpi_100"); got != "unrestricted" {
		t.Fatalf("matching active system access mode = %q, want unrestricted", got)
	}
	if got := runtime.currentAccessMode("gpd_100"); got != "restricted" {
		t.Fatalf("mismatched system access mode = %q, want restricted", got)
	}
}
