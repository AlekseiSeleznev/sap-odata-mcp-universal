package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zmcp/odata-mcp/internal/models"
)

type fakeProvider struct{}

func (fakeProvider) Status(ctx context.Context) (*models.DashboardConnectionStatus, error) {
	return &models.DashboardConnectionStatus{
		ActiveDefault:    "dev-s4",
		Connected:        true,
		Transport:        "streamable-http",
		HTTPAddr:         "localhost:3000",
		TotalConnections: 1,
	}, nil
}

func (fakeProvider) Connections(ctx context.Context) ([]models.DashboardConnection, error) {
	return []models.DashboardConnection{
		{
			Name:           "dev-s4",
			SystemName:     "S4 DEV",
			ServiceURL:     "https://sap.example.test/sap/opu/odata/sap/API_TEST_SRV/",
			SafeServiceURL: "https://sap.example.test/sap/opu/odata/sap/API_TEST_SRV/",
			Client:         "100",
			Username:       "demo",
			AccessMode:     "restricted",
			Connected:      true,
		},
	}, nil
}

func (fakeProvider) Connect(ctx context.Context, req models.DashboardConnectionUpsertRequest) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "connected", Status: models.DashboardConnectionStatus{ActiveDefault: req.Name, Connected: true}}, nil
}

func (fakeProvider) Disconnect(ctx context.Context, name string) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "disconnected", Status: models.DashboardConnectionStatus{}}, nil
}

func (fakeProvider) Edit(ctx context.Context, req models.DashboardConnectionEditRequest) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "saved", Status: models.DashboardConnectionStatus{ActiveDefault: req.Name, Connected: true}}, nil
}

func (fakeProvider) Switch(ctx context.Context, name string) (*models.DashboardMutationResult, error) {
	return &models.DashboardMutationResult{OK: true, Message: "switched", Status: models.DashboardConnectionStatus{ActiveDefault: name, Connected: true}}, nil
}

func (fakeProvider) DocsContext(ctx context.Context) (*models.DashboardDocumentationContext, error) {
	return &models.DashboardDocumentationContext{
		Transport:            "streamable-http",
		HTTPAddr:             "localhost:3000",
		MCPPath:              "/mcp",
		HealthPath:           "/health",
		DashboardPath:        "/dashboard",
		DocumentationPath:    "/dashboard/docs",
		StatusPath:           "/api/status",
		ListPath:             "/api/databases",
		ConnectPath:          "/api/connect",
		DisconnectPath:       "/api/disconnect",
		EditPath:             "/api/edit",
		SwitchPath:           "/api/switch",
		StateFile:            "/tmp/odata-state.json",
		SupportsEmptyStartup: true,
		ActiveConnection:     "dev-s4",
		TotalConnections:     1,
	}, nil
}

func (fakeProvider) RestoreActiveConnection(ctx context.Context) error {
	return nil
}

func TestDashboardRoutes(t *testing.T) {
	mux := http.NewServeMux()
	New(fakeProvider{}).RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected redirect for root, got %d", rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/dashboard" {
		t.Fatalf("expected root redirect to /dashboard, got %q", location)
	}

	req = httptest.NewRequest(http.MethodGet, "/dashboard?lang=ru", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for dashboard, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "sap-odata-mcp-universal") {
		t.Fatalf("dashboard page missing expected title")
	}
	if !strings.Contains(rec.Body.String(), "Системы SAP") {
		t.Fatalf("dashboard page missing localized heading")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for status, got %d", rec.Code)
	}
	var status models.DashboardConnectionStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("invalid status JSON: %v", err)
	}
	if status.ActiveDefault != "dev-s4" {
		t.Fatalf("unexpected active default: %q", status.ActiveDefault)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/databases", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for connections, got %d", rec.Code)
	}
	var items []models.DashboardConnection
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("invalid connections JSON: %v", err)
	}
	if len(items) != 1 || items[0].Name != "dev-s4" {
		t.Fatalf("unexpected connections payload: %+v", items)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/connect", strings.NewReader(`{"name":"qa-s4","system_name":"S4 QA","service_url":"https://sap.example.test/qa","username":"demo","password":"secret","access_mode":"restricted"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for connect, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/edit", strings.NewReader(`{"old_name":"dev-s4","name":"dev-s4","system_name":"S4 DEV","service_url":"https://sap.example.test/dev","username":"demo","password":"","access_mode":"restricted"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for edit, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/switch", strings.NewReader(`{"name":"dev-s4"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for switch, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/disconnect", strings.NewReader(`{"name":"dev-s4"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for disconnect, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/dashboard/docs?lang=en", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for docs, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Connection parameters") {
		t.Fatalf("docs page missing detailed documentation content")
	}
}
