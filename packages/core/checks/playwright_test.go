package checks_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"status-page/packages/core/checks"
	"status-page/packages/core/domain"
)

func TestPlaywrightChecker_Registration(t *testing.T) {
	c, ok := checks.Get("playwright")
	if !ok {
		t.Fatal("expected 'playwright' checker to be registered in DefaultRegistry")
	}
	if c.Type() != "playwright" {
		t.Fatalf("expected type 'playwright', got %s", c.Type())
	}
}

func TestPlaywrightChecker_Execute_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req checks.PlaywrightRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		if req.Target != "https://example.com" {
			t.Errorf("expected target https://example.com, got %s", req.Target)
		}

		res := checks.PlaywrightRunResponse{
			Success:   true,
			State:     domain.StateOperational,
			LatencyMs: 350,
			Message:   "Page load nominal (350ms, HTTP 200)",
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer ts.Close()

	checker := checks.NewPlaywrightChecker(ts.Client(), ts.URL)
	def := domain.CheckDefinition{
		ID:      "pw-1",
		Feature: "ui",
		Type:    "playwright",
		Target:  "https://example.com",
	}

	result, err := checker.Execute(context.Background(), def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected Success=true")
	}
	if result.State != domain.StateOperational {
		t.Errorf("expected State=operational, got %s", result.State)
	}
	if result.Message != "Page load nominal (350ms, HTTP 200)" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestPlaywrightChecker_Execute_Degraded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := checks.PlaywrightRunResponse{
			Success:   true,
			State:     domain.StateDegraded,
			LatencyMs: 4200,
			Message:   "Page load slow (4200ms)",
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer ts.Close()

	checker := checks.NewPlaywrightChecker(ts.Client(), ts.URL)
	def := domain.CheckDefinition{
		ID:      "pw-2",
		Feature: "ui",
		Type:    "playwright",
		Target:  "https://example.com/slow",
	}

	result, err := checker.Execute(context.Background(), def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != domain.StateDegraded {
		t.Errorf("expected State=degraded, got %s", result.State)
	}
}

func TestPlaywrightChecker_Execute_OutageOnServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"success":false,"state":"outage","latencyMs":0,"message":"Playwright check error: net::ERR_CONNECTION_REFUSED"}`))
	}))
	defer ts.Close()

	checker := checks.NewPlaywrightChecker(ts.Client(), ts.URL)
	def := domain.CheckDefinition{
		ID:      "pw-3",
		Feature: "ui",
		Type:    "playwright",
		Target:  "https://invalid.test",
	}

	result, err := checker.Execute(context.Background(), def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Errorf("expected Success=false")
	}
	if result.State != domain.StateOutage {
		t.Errorf("expected State=outage, got %s", result.State)
	}
}

func TestPlaywrightChecker_Execute_ConnectionError(t *testing.T) {
	checker := checks.NewPlaywrightChecker(nil, "http://127.0.0.1:59999/run")
	def := domain.CheckDefinition{
		ID:      "pw-4",
		Feature: "ui",
		Type:    "playwright",
		Target:  "https://example.com",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	result, err := checker.Execute(ctx, def)
	if err == nil {
		t.Errorf("expected connection error")
	}
	if result.Success {
		t.Errorf("expected Success=false")
	}
	if result.State != domain.StateOutage {
		t.Errorf("expected State=outage, got %s", result.State)
	}
}
