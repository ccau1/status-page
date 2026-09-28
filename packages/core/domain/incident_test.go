package domain_test

import (
	"testing"

	"status-page/packages/core/domain"
)

func TestIncident_AppliesToProduct(t *testing.T) {
	// Global incident
	globalInc := domain.Incident{
		ID:    "inc-1",
		Title: "Platform-wide DNS degradation",
	}
	if !globalInc.AppliesToProduct("cloud-api") {
		t.Errorf("global incident should apply to any product")
	}

	// Multi-product incident
	multiInc := domain.Incident{
		ID:       "inc-2",
		Title:    "Edge routing disruption",
		Products: []string{"cloud-api", "auth-service"},
	}
	if !multiInc.AppliesToProduct("cloud-api") {
		t.Errorf("multi-product incident should apply to cloud-api")
	}
	if !multiInc.AppliesToProduct("auth-service") {
		t.Errorf("multi-product incident should apply to auth-service")
	}
	if multiInc.AppliesToProduct("billing-engine") {
		t.Errorf("multi-product incident should NOT apply to billing-engine")
	}

	// Legacy single product incident
	singleInc := domain.Incident{
		ID:      "inc-3",
		Product: "billing-engine",
	}
	if !singleInc.AppliesToProduct("billing-engine") {
		t.Errorf("single product incident should apply to billing-engine")
	}
	if singleInc.AppliesToProduct("cloud-api") {
		t.Errorf("single product incident should NOT apply to cloud-api")
	}
}

func TestIncident_AppliesToCheck(t *testing.T) {
	inc := domain.Incident{
		ID:       "inc-check",
		CheckIDs: []string{"chk-gateway-1", "chk-rate-1"},
		Features: []string{"payments"},
	}

	if !inc.AppliesToCheck("chk-gateway-1", "api-gateway") {
		t.Errorf("should match linked check ID")
	}
	if !inc.AppliesToCheck("chk-other", "payments") {
		t.Errorf("should match linked feature ID")
	}
	if inc.AppliesToCheck("chk-other", "other-feature") {
		t.Errorf("should NOT match unrelated check and feature")
	}
}
