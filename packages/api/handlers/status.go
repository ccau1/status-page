package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"status-page/packages/core/domain"
	"status-page/packages/core/ports"
)

type StatusHandler struct {
	storage ports.StoragePort
}

func NewStatusHandler(storage ports.StoragePort) *StatusHandler {
	return &StatusHandler{
		storage: storage,
	}
}

// Health handles healthcheck requests.
func (h *StatusHandler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// ListAll handles GET /api/status
func (h *StatusHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.storage.ListAllStatuses(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query statuses: "+err.Error())
		return
	}
	if statuses == nil {
		statuses = []domain.Status{}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": statuses,
		"count": len(statuses),
	})
}

// ListByTenant handles GET /api/status/{tenant}
func (h *StatusHandler) ListByTenant(w http.ResponseWriter, r *http.Request) {
	tenant := r.PathValue("tenant")
	if tenant == "" {
		respondError(w, http.StatusBadRequest, "tenant parameter is required")
		return
	}

	statuses, err := h.storage.ListStatusesByTenant(r.Context(), tenant)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query tenant statuses: "+err.Error())
		return
	}
	if statuses == nil {
		statuses = []domain.Status{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tenant": tenant,
		"items":  statuses,
		"count":  len(statuses),
	})
}

// GetProductStatus handles GET /api/status/{tenant}/{product}
func (h *StatusHandler) GetProductStatus(w http.ResponseWriter, r *http.Request) {
	tenant := r.PathValue("tenant")
	product := r.PathValue("product")

	if tenant == "" || product == "" {
		respondError(w, http.StatusBadRequest, "both tenant and product parameters are required")
		return
	}

	status, err := h.storage.GetStatus(r.Context(), tenant, product)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query status: "+err.Error())
		return
	}
	if status == nil {
		respondError(w, http.StatusNotFound, "status not found for tenant: "+tenant+" product: "+product)
		return
	}

	respondJSON(w, http.StatusOK, status)
}

// ListIncidents handles GET /api/incidents and GET /api/incidents/{tenant}
func (h *StatusHandler) ListIncidents(w http.ResponseWriter, r *http.Request) {
	tenant := r.PathValue("tenant")
	if tenant == "" {
		tenant = "default"
	}

	incidents, err := h.storage.GetActiveIncidents(r.Context(), tenant)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query incidents: "+err.Error())
		return
	}
	if incidents == nil {
		incidents = []domain.Incident{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tenant":    tenant,
		"incidents": incidents,
		"count":     len(incidents),
	})
}

// CreateOrUpdateIncident handles POST /api/incidents
func (h *StatusHandler) CreateOrUpdateIncident(w http.ResponseWriter, r *http.Request) {
	var inc domain.Incident
	if err := json.NewDecoder(r.Body).Decode(&inc); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if inc.ID == "" || inc.Title == "" {
		respondError(w, http.StatusBadRequest, "id and title are required")
		return
	}
	if inc.Tenant == "" {
		inc.Tenant = "*"
	}
	if inc.Severity == "" {
		inc.Severity = domain.SeverityMajor
	}
	if inc.State == "" {
		inc.State = domain.IncidentInvestigating
	}
	if inc.CreatedAt.IsZero() {
		inc.CreatedAt = time.Now().UTC()
	}
	inc.UpdatedAt = time.Now().UTC()

	if err := h.storage.SaveIncident(r.Context(), &inc); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save incident: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, inc)
}

// DeleteIncident handles DELETE /api/incidents/{id}
func (h *StatusHandler) DeleteIncident(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "incident id parameter is required")
		return
	}

	// Deactivate the incident
	inc := domain.Incident{
		ID:        id,
		Tenant:    "*",
		Title:     "Archived Incident",
		Severity:  domain.SeverityInfo,
		State:     domain.IncidentResolved,
		Active:    false,
		UpdatedAt: time.Now().UTC(),
	}

	if err := h.storage.SaveIncident(r.Context(), &inc); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to deactivate incident: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "deactivated",
		"id":     id,
	})
}

func respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, map[string]string{
		"error": message,
	})
}
