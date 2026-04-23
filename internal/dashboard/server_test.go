package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/models"
)

type fakeProvider struct{}

func (fakeProvider) Status(ctx context.Context) (*models.DashboardHierarchyStatus, error) {
	return &models.DashboardHierarchyStatus{
		ActiveSystemID:   "s4d-100",
		ActiveSystemName: "S4D",
		Connected:        true,
		Transport:        "streamable-http",
		HTTPAddr:         "localhost:3000",
		TotalSystems:     1,
		TotalServices:    2,
		TotalEntities:    1,
		TotalOperations:  2,
	}, nil
}

func (fakeProvider) Systems(ctx context.Context) ([]models.DashboardSystem, error) {
	return []models.DashboardSystem{
		{
			ID:         "s4d-100",
			Name:       "S4D",
			BaseURL:    "http://s4d.msgplaut.com:8000",
			Client:     "100",
			Username:   "demo",
			AccessMode: "unrestricted",
			Connected:  true,
			Active:     true,
			Services: []models.DashboardService{
				{ID: "materials-read", Name: "materials-read", ServiceURL: "http://host/MMIM_MATERIAL_DATA_SRV/", SafeServiceURL: "http://host/MMIM_MATERIAL_DATA_SRV/"},
			},
			Entities: []models.DashboardEntity{
				{
					ID:    "materials",
					Label: "Materials",
					Operations: []models.DashboardOperation{
						{ID: "materials-get", Verb: "get", ServiceID: "materials-read", ServiceName: "materials-read", EntitySet: "MaterialHeaders", ToolName: "materials_get_for_s4d-100", Mode: "generated", Enabled: true},
					},
				},
			},
		},
	}, nil
}

func (fakeProvider) SaveSystem(ctx context.Context, req models.DashboardSystemUpsertRequest) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "saved"}, nil
}
func (fakeProvider) DeleteSystem(ctx context.Context, id string) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "deleted"}, nil
}
func (fakeProvider) ActivateSystem(ctx context.Context, id string) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "activated"}, nil
}
func (fakeProvider) SaveService(ctx context.Context, req models.DashboardServiceUpsertRequest) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "saved"}, nil
}
func (fakeProvider) DeleteService(ctx context.Context, req models.DashboardDeleteRequest) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "deleted"}, nil
}
func (fakeProvider) SaveEntity(ctx context.Context, req models.DashboardEntityUpsertRequest) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "saved"}, nil
}
func (fakeProvider) DeleteEntity(ctx context.Context, req models.DashboardDeleteRequest) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "deleted"}, nil
}
func (fakeProvider) SaveOperation(ctx context.Context, req models.DashboardOperationUpsertRequest) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "saved"}, nil
}
func (fakeProvider) DeleteOperation(ctx context.Context, req models.DashboardDeleteRequest) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "deleted"}, nil
}
func (fakeProvider) DiscoverService(ctx context.Context, systemID, serviceID string) (*models.DashboardServiceDiscovery, error) {
	return &models.DashboardServiceDiscovery{
		SystemID:    systemID,
		ServiceID:   serviceID,
		ServiceName: "materials-read",
		EntitySets: []models.EntitySetSummary{
			{Name: "MaterialHeaders", EntityType: "MaterialHeader", Creatable: false},
		},
	}, nil
}
func (fakeProvider) DocsContext(ctx context.Context) (*models.DashboardDocumentationContext, error) {
	return &models.DashboardDocumentationContext{
		Transport:           "streamable-http",
		HTTPAddr:            "localhost:3000",
		MCPPath:             "/mcp",
		StatusPath:          "/api/status",
		SystemsPath:         "/api/systems",
		SaveSystemPath:      "/api/system/save",
		DeleteSystemPath:    "/api/system/delete",
		ActivateSystemPath:  "/api/system/activate",
		SaveServicePath:     "/api/service/save",
		DeleteServicePath:   "/api/service/delete",
		SaveEntityPath:      "/api/entity/save",
		DeleteEntityPath:    "/api/entity/delete",
		SaveOperationPath:   "/api/operation/save",
		DeleteOperationPath: "/api/operation/delete",
		DiscoveryPath:       "/api/service/discover",
		StateFile:           "/tmp/state.json",
	}, nil
}
func (fakeProvider) RestoreActiveConnection(ctx context.Context) error { return nil }

func TestDashboardRoutes(t *testing.T) {
	mux := http.NewServeMux()
	New(fakeProvider{}).RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected redirect for root, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/dashboard?lang=ru", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for dashboard, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "sap-odata-mcp-universal") || !strings.Contains(rec.Body.String(), "Системы и сущности") {
		t.Fatalf("dashboard page missing expected content")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for status, got %d", rec.Code)
	}
	var status models.DashboardHierarchyStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("invalid status JSON: %v", err)
	}
	if status.ActiveSystemID != "s4d-100" {
		t.Fatalf("unexpected active system id: %q", status.ActiveSystemID)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/systems", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for systems, got %d", rec.Code)
	}
	var systems []models.DashboardSystem
	if err := json.Unmarshal(rec.Body.Bytes(), &systems); err != nil {
		t.Fatalf("invalid systems JSON: %v", err)
	}
	if len(systems) != 1 || systems[0].ID != "s4d-100" {
		t.Fatalf("unexpected systems payload: %+v", systems)
	}

	for _, tc := range []struct {
		path string
		body string
	}{
		{"/api/system/save", `{"name":"S4D","username":"demo","password":"secret","access_mode":"unrestricted"}`},
		{"/api/system/activate", `{"system_id":"s4d-100"}`},
		{"/api/service/save", `{"system_id":"s4d-100","name":"read","service_url":"http://host/service/"}`},
		{"/api/entity/save", `{"system_id":"s4d-100","label":"Materials"}`},
		{"/api/operation/save", `{"system_id":"s4d-100","entity_id":"materials","verb":"get","service_id":"materials-read","entity_set":"MaterialHeaders","enabled":true}`},
	} {
		req = httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", tc.path, rec.Code)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/api/service/discover?system_id=s4d-100&service_id=materials-read", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for discovery, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/dashboard/docs?lang=en", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for docs, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "System") || !strings.Contains(rec.Body.String(), "Dashboard HTTP API") {
		t.Fatalf("docs page missing hierarchical documentation content")
	}
}
