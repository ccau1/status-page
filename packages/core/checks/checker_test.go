package checks_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"status-page/packages/core/checks"
	"status-page/packages/core/domain"
)

func TestHTTPChecker_Execute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	ctx := context.Background()

	// 1. Success case with expected body
	res := checks.ExecuteCheck(ctx, nil, domain.CheckDefinition{
		ID:           "test-http-ok",
		Feature:      "api",
		Type:         "http",
		Target:       server.URL + "/health",
		ExpectedBody: "status",
	})
	if !res.Success {
		t.Fatalf("expected success, got failure: %s", res.Message)
	}
	if res.State != domain.StateOperational {
		t.Errorf("expected StateOperational, got %v", res.State)
	}

	// 2. 500 error case
	resErr := checks.ExecuteCheck(ctx, nil, domain.CheckDefinition{
		ID:      "test-http-err",
		Feature: "api",
		Type:    "http",
		Target:  server.URL + "/error",
	})
	if resErr.Success {
		t.Fatalf("expected check to fail on 500 status")
	}
	if resErr.State != domain.StateOutage {
		t.Errorf("expected StateOutage, got %v", resErr.State)
	}
}

func TestSyntheticChecker(t *testing.T) {
	ctx := context.Background()

	res := checks.ExecuteCheck(ctx, nil, domain.CheckDefinition{
		ID:      "synth-degraded",
		Feature: "billing",
		Type:    "synthetic",
		Target:  "simulated-billing-gateway",
		Parameters: map[string]string{
			"simulated_state": "degraded",
			"message":         "high payment queue latency",
		},
	})

	if res.State != domain.StateDegraded {
		t.Errorf("expected degraded state, got %v", res.State)
	}
}

func TestChecker_UnknownType(t *testing.T) {
	ctx := context.Background()

	res := checks.ExecuteCheck(ctx, nil, domain.CheckDefinition{
		ID:      "unknown-check",
		Feature: "billing",
		Type:    "non-existent-type",
		Target:  "foo",
	})

	if res.Success {
		t.Fatalf("expected failure for unknown check type")
	}
	if res.State != domain.StateOutage {
		t.Errorf("expected StateOutage for unknown check type, got %v", res.State)
	}
}

func TestRegistry_CustomChecker(t *testing.T) {
	reg := checks.NewRegistry()

	custom := &customEchoChecker{}
	reg.Register(custom)

	c, ok := reg.Get("custom-echo")
	if !ok || c.Type() != "custom-echo" {
		t.Fatalf("failed to retrieve custom registered checker")
	}

	res := checks.ExecuteCheck(context.Background(), reg, domain.CheckDefinition{
		ID:      "custom-1",
		Feature: "plugins",
		Type:    "custom-echo",
		Target:  "echo-target",
	})
	if !res.Success {
		t.Fatalf("expected custom check success, got: %s", res.Message)
	}
}

type customEchoChecker struct{}

func (c *customEchoChecker) Type() string { return "custom-echo" }
func (c *customEchoChecker) Execute(ctx context.Context, def domain.CheckDefinition) (domain.CheckResult, error) {
	return domain.CheckResult{
		Success:   true,
		State:     domain.StateOperational,
		Message:   "echo: " + def.Target,
		Timestamp: time.Now().UTC(),
	}, nil
}
