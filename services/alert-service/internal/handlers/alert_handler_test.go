package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"alert-service/internal/models"

	"github.com/go-chi/chi/v5"
)

func TestHealth(t *testing.T) {
	handler := &AlertHandler{}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["service"] != "alert-service" {
		t.Fatalf("expected service alert-service, got %s", response["service"])
	}

	if response["status"] != "healthy" {
		t.Fatalf("expected status healthy, got %s", response["status"])
	}
}

func TestCreateIncident(t *testing.T) {
	repo := &fakeAlertRepository{}
	handler := NewAlertHandler(repo)

	body := `{
		"alert_id": 123,
		"asset_id": "asset-101",
		"title": "Critical high temperature",
		"description": "Temperature crossed critical threshold",
		"severity": "CRITICAL"
	}`

	req := httptest.NewRequest(http.MethodPost, "/incidents", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.CreateIncident(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	if len(repo.incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(repo.incidents))
	}

	if repo.incidents[0].Status != "OPEN" {
		t.Fatalf("expected incident status OPEN, got %s", repo.incidents[0].Status)
	}

	if len(repo.history) != 1 || repo.history[0].Action != "CREATED" {
		t.Fatalf("expected CREATED history entry, got %+v", repo.history)
	}
}

func TestListIncidents(t *testing.T) {
	repo := &fakeAlertRepository{}
	operator := "operator@example.com"
	repo.incidents = []models.Incident{
		{ID: 1, AssetID: "asset-101", Severity: "CRITICAL", Status: "OPEN"},
		{ID: 2, AssetID: "asset-102", Severity: "HIGH", Status: "ASSIGNED", AssignedTo: &operator},
	}
	handler := NewAlertHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/incidents?status=ASSIGNED&assigned_to=operator@example.com", nil)
	rec := httptest.NewRecorder()

	handler.ListIncidents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var incidents []models.Incident
	if err := json.NewDecoder(rec.Body).Decode(&incidents); err != nil {
		t.Fatalf("failed to decode incidents: %v", err)
	}

	if len(incidents) != 1 || incidents[0].ID != 2 {
		t.Fatalf("expected assigned incident #2, got %+v", incidents)
	}
}

func TestIncidentLifecycleCreatesHistory(t *testing.T) {
	repo := &fakeAlertRepository{}
	handler := NewAlertHandler(repo)
	incident := models.Incident{
		AssetID:     "asset-101",
		Title:       "Critical high temperature",
		Description: "Temperature crossed critical threshold",
		Severity:    "CRITICAL",
	}
	if err := repo.CreateIncident(nil, &incident, "system", "created"); err != nil {
		t.Fatalf("failed to seed incident: %v", err)
	}

	router := chi.NewRouter()
	router.Put("/incidents/{id}/assign", handler.AssignIncident)
	router.Put("/incidents/{id}/acknowledge", handler.AcknowledgeIncident)
	router.Put("/incidents/{id}/resolve", handler.ResolveIncident)
	router.Put("/incidents/{id}/close", handler.CloseIncident)
	router.Get("/incidents/{id}/history", handler.GetIncidentHistory)

	assertIncidentAction(t, router, http.MethodPut, "/incidents/1/assign", `{
		"assigned_to": "operator@example.com",
		"actor": "admin@example.com",
		"comment": "Assigning to shift operator"
	}`, "ASSIGNED")

	assertIncidentAction(t, router, http.MethodPut, "/incidents/1/acknowledge", `{
		"actor": "operator@example.com",
		"comment": "Investigating issue"
	}`, "ACKNOWLEDGED")

	resolved := assertIncidentAction(t, router, http.MethodPut, "/incidents/1/resolve", `{
		"actor": "operator@example.com",
		"resolution_note": "Cooling system restarted"
	}`, "RESOLVED")

	if resolved.ResolutionNote == nil || *resolved.ResolutionNote != "Cooling system restarted" {
		t.Fatalf("expected resolution note to be stored, got %+v", resolved.ResolutionNote)
	}

	assertIncidentAction(t, router, http.MethodPut, "/incidents/1/close", `{
		"actor": "admin@example.com",
		"comment": "Verified and closed"
	}`, "CLOSED")

	req := httptest.NewRequest(http.MethodGet, "/incidents/1/history", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected history status %d, got %d", http.StatusOK, rec.Code)
	}

	var history []models.IncidentHistory
	if err := json.NewDecoder(rec.Body).Decode(&history); err != nil {
		t.Fatalf("failed to decode history: %v", err)
	}

	if len(history) != 5 {
		t.Fatalf("expected 5 history entries, got %d", len(history))
	}
}

func TestCriticalAlertAutomaticallyCreatesIncident(t *testing.T) {
	repo := &fakeAlertRepository{}
	handler := NewAlertHandler(repo)

	body := `{
		"assetId": "asset-101",
		"name": "Critical high temperature",
		"severity": "CRITICAL",
		"message": "Temperature crossed critical threshold"
	}`

	req := httptest.NewRequest(http.MethodPost, "/alerts", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.CreateAlert(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	if len(repo.incidents) != 1 {
		t.Fatalf("expected critical alert to create 1 incident, got %d", len(repo.incidents))
	}

	if repo.incidents[0].AlertID == nil || *repo.incidents[0].AlertID != 1 {
		t.Fatalf("expected incident to reference alert 1, got %+v", repo.incidents[0].AlertID)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/alerts", strings.NewReader(body))
	secondRec := httptest.NewRecorder()

	handler.CreateAlert(secondRec, secondReq)

	if secondRec.Code != http.StatusCreated {
		t.Fatalf("expected second status %d, got %d", http.StatusCreated, secondRec.Code)
	}

	if len(repo.incidents) != 1 {
		t.Fatalf("expected duplicate critical alert to keep 1 active incident, got %d", len(repo.incidents))
	}
}

func assertIncidentAction(t *testing.T, router http.Handler, method string, path string, body string, expectedStatus string) models.Incident {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected action status %d, got %d", http.StatusOK, rec.Code)
	}

	var incident models.Incident
	if err := json.NewDecoder(rec.Body).Decode(&incident); err != nil {
		t.Fatalf("failed to decode incident: %v", err)
	}

	if incident.Status != expectedStatus {
		t.Fatalf("expected status %s, got %s", expectedStatus, incident.Status)
	}

	return incident
}
