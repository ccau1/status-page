package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"status-page/packages/api/handlers"
	"status-page/packages/core/adapters/sqlite"
	"status-page/packages/core/domain"
)

func TestStatusHandlers(t *testing.T) {
	tempDB := t.TempDir() + "/test_api.db"
	store, err := sqlite.New(tempDB)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.Init(ctx); err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	// Seed sample status
	sampleStatus := &domain.Status{
		Tenant:       "acme",
		Product:      "payments",
		CurrentState: domain.StateOperational,
		Message:      "All systems functional",
		LastUpdated:  time.Now().UTC(),
		Features: map[string]domain.FeatureStatus{
			"checkout": {
				ID:    "checkout",
				Name:  "Checkout Flow",
				State: domain.StateOperational,
			},
		},
	}
	if err := store.SaveStatus(ctx, sampleStatus); err != nil {
		t.Fatalf("failed to save sample status: %v", err)
	}

	h := handlers.NewStatusHandler(store)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/status", h.ListAll)
	mux.HandleFunc("GET /api/status/{tenant}", h.ListByTenant)
	mux.HandleFunc("GET /api/status/{tenant}/{product}", h.GetProductStatus)
	mux.HandleFunc("GET /api/incidents", h.ListIncidents)
	mux.HandleFunc("GET /api/incidents/{tenant}", h.ListIncidents)
	mux.HandleFunc("POST /api/incidents", h.CreateOrUpdateIncident)

	// Test 1: GET /health
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("health endpoint returned %d, expected 200", rec.Code)
	}

	// Test 2: GET /api/status
	req = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status endpoint returned %d, expected 200", rec.Code)
	}
	var allRes map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &allRes); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if int(allRes["count"].(float64)) != 1 {
		t.Errorf("expected count 1, got %v", allRes["count"])
	}

	// Test 3: GET /api/status/acme
	req = httptest.NewRequest(http.MethodGet, "/api/status/acme", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("tenant endpoint returned %d, expected 200", rec.Code)
	}

	// Test 4: GET /api/status/acme/payments
	req = httptest.NewRequest(http.MethodGet, "/api/status/acme/payments", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("product endpoint returned %d, expected 200", rec.Code)
	}

	// Test 5: GET /api/status/acme/unknown (404)
	req = httptest.NewRequest(http.MethodGet, "/api/status/acme/unknown", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("non-existent product returned %d, expected 404", rec.Code)
	}

	// Test 6: POST /api/incidents (Create active incident banner)
	incidentPayload := `{"id":"inc-101","tenant":"acme","title":"Database Latency","severity":"major","state":"investigating","message":"Engineers are investigating DB latency","active":true}`
	req = httptest.NewRequest(http.MethodPost, "/api/incidents", strings.NewReader(incidentPayload))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("create incident returned %d, expected 200: %s", rec.Code, rec.Body.String())
	}

	// Test 7: GET /api/incidents/acme
	req = httptest.NewRequest(http.MethodGet, "/api/incidents/acme", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("list incidents returned %d, expected 200", rec.Code)
	}
	var incRes map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &incRes); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if int(incRes["count"].(float64)) != 1 {
		t.Errorf("expected count 1, got %v", incRes["count"])
	}
}
