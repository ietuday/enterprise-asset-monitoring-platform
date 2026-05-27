package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"alert-service/internal/models"

	"github.com/jackc/pgx/v5"
)

type fakeAlertRepository struct {
	createdAlerts  []models.Alert
	resolvedAlerts []models.Alert
	incidents      []models.Incident
	history        []models.IncidentHistory
}

func (f *fakeAlertRepository) Create(ctx context.Context, alert *models.Alert) error {
	for _, existing := range f.createdAlerts {
		if existing.AssetID == alert.AssetID &&
			existing.Name == alert.Name &&
			(existing.Status == "OPEN" || existing.Status == "ACKNOWLEDGED") {
			*alert = existing
			return nil
		}
	}

	alert.ID = int64(len(f.createdAlerts) + 1)
	f.createdAlerts = append(f.createdAlerts, *alert)
	return nil
}

func (f *fakeAlertRepository) List(ctx context.Context) ([]models.Alert, error) {
	return f.createdAlerts, nil
}

func (f *fakeAlertRepository) GetByID(ctx context.Context, id string) (*models.Alert, error) {
	return nil, nil
}

func (f *fakeAlertRepository) Acknowledge(ctx context.Context, id string) (*models.Alert, error) {
	return nil, nil
}

func (f *fakeAlertRepository) Resolve(ctx context.Context, id string) (*models.Alert, error) {
	return nil, nil
}

func (f *fakeAlertRepository) ResolveActiveByAssetAndName(ctx context.Context, assetID string, name string) (*models.Alert, error) {
	alert := &models.Alert{
		ID:       1,
		AssetID:  assetID,
		Name:     name,
		Severity: "CRITICAL",
		Status:   "RESOLVED",
		Message:  "resolved from test",
	}

	f.resolvedAlerts = append(f.resolvedAlerts, *alert)

	return alert, nil
}

func (f *fakeAlertRepository) CreateIncident(ctx context.Context, incident *models.Incident, actor string, comment string) error {
	if incident.AlertID != nil {
		for _, existing := range f.incidents {
			if existing.AlertID != nil && *existing.AlertID == *incident.AlertID && existing.Status != "RESOLVED" && existing.Status != "CLOSED" {
				*incident = existing
				return nil
			}
		}
	}

	now := time.Now()
	incident.ID = int64(len(f.incidents) + 1)
	incident.Status = "OPEN"
	incident.CreatedAt = now
	incident.UpdatedAt = now
	f.incidents = append(f.incidents, *incident)
	f.history = append(f.history, models.IncidentHistory{
		ID:         int64(len(f.history) + 1),
		IncidentID: incident.ID,
		Action:     "CREATED",
		NewStatus:  "OPEN",
		Actor:      actor,
		Comment:    comment,
		CreatedAt:  now,
	})
	return nil
}

func (f *fakeAlertRepository) ListIncidents(ctx context.Context, filters models.IncidentFilters) ([]models.Incident, error) {
	items := make([]models.Incident, 0)
	for _, incident := range f.incidents {
		if filters.Status != "" && incident.Status != filters.Status {
			continue
		}
		if filters.Severity != "" && incident.Severity != filters.Severity {
			continue
		}
		if filters.AssignedTo != "" && (incident.AssignedTo == nil || *incident.AssignedTo != filters.AssignedTo) {
			continue
		}
		items = append(items, incident)
	}
	return items, nil
}

func (f *fakeAlertRepository) GetIncidentByID(ctx context.Context, id string) (*models.Incident, error) {
	for i := range f.incidents {
		if id == stringID(f.incidents[i].ID) {
			return &f.incidents[i], nil
		}
	}
	return nil, pgx.ErrNoRows
}

func (f *fakeAlertRepository) AssignIncident(ctx context.Context, id string, assignedTo string, actor string, comment string) (*models.Incident, error) {
	return f.updateIncident(id, "ASSIGNED", actor, comment, func(incident *models.Incident) {
		incident.AssignedTo = &assignedTo
	})
}

func (f *fakeAlertRepository) AcknowledgeIncident(ctx context.Context, id string, actor string, comment string) (*models.Incident, error) {
	return f.updateIncident(id, "ACKNOWLEDGED", actor, comment, func(incident *models.Incident) {
		now := time.Now()
		incident.AcknowledgedAt = &now
	})
}

func (f *fakeAlertRepository) ResolveIncident(ctx context.Context, id string, actor string, resolutionNote string) (*models.Incident, error) {
	return f.updateIncident(id, "RESOLVED", actor, resolutionNote, func(incident *models.Incident) {
		now := time.Now()
		incident.ResolvedAt = &now
		incident.ResolutionNote = &resolutionNote
	})
}

func (f *fakeAlertRepository) CloseIncident(ctx context.Context, id string, actor string, comment string) (*models.Incident, error) {
	return f.updateIncident(id, "CLOSED", actor, comment, func(incident *models.Incident) {
		now := time.Now()
		incident.ClosedAt = &now
	})
}

func (f *fakeAlertRepository) GetIncidentHistory(ctx context.Context, incidentID string) ([]models.IncidentHistory, error) {
	items := make([]models.IncidentHistory, 0)
	for _, item := range f.history {
		if incidentID == stringID(item.IncidentID) {
			items = append(items, item)
		}
	}
	return items, nil
}

func (f *fakeAlertRepository) AddIncidentHistory(ctx context.Context, history *models.IncidentHistory) error {
	history.ID = int64(len(f.history) + 1)
	history.CreatedAt = time.Now()
	f.history = append(f.history, *history)
	return nil
}

func (f *fakeAlertRepository) updateIncident(id string, status string, actor string, comment string, mutate func(*models.Incident)) (*models.Incident, error) {
	for i := range f.incidents {
		if id != stringID(f.incidents[i].ID) {
			continue
		}

		oldStatus := f.incidents[i].Status
		f.incidents[i].Status = status
		f.incidents[i].UpdatedAt = time.Now()
		mutate(&f.incidents[i])
		f.history = append(f.history, models.IncidentHistory{
			ID:         int64(len(f.history) + 1),
			IncidentID: f.incidents[i].ID,
			Action:     status,
			OldStatus:  &oldStatus,
			NewStatus:  status,
			Actor:      actor,
			Comment:    comment,
			CreatedAt:  time.Now(),
		})

		return &f.incidents[i], nil
	}

	return nil, pgx.ErrNoRows
}

func stringID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func TestAlertmanagerWebhookCreatesAlertOnFiring(t *testing.T) {
	repo := &fakeAlertRepository{}
	handler := NewAlertHandler(repo)

	body := `{
		"status": "firing",
		"alerts": [
			{
				"status": "firing",
				"labels": {
					"alertname": "HighTemperature",
					"asset_id": "motor-101",
					"severity": "critical"
				},
				"annotations": {
					"alert_name": "High Temperature",
					"description": "Asset motor-101 temperature is above threshold"
				}
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/alerts/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.AlertmanagerWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if len(repo.createdAlerts) != 1 {
		t.Fatalf("expected 1 created alert, got %d", len(repo.createdAlerts))
	}

	alert := repo.createdAlerts[0]

	if alert.AssetID != "motor-101" {
		t.Fatalf("expected asset motor-101, got %s", alert.AssetID)
	}

	if alert.Name != "High Temperature" {
		t.Fatalf("expected High Temperature alert, got %s", alert.Name)
	}

	if alert.Severity != "CRITICAL" {
		t.Fatalf("expected severity CRITICAL, got %s", alert.Severity)
	}

	if alert.Status != "OPEN" {
		t.Fatalf("expected status OPEN, got %s", alert.Status)
	}
}

func TestAlertmanagerWebhookResolvesAlertOnResolved(t *testing.T) {
	repo := &fakeAlertRepository{}
	handler := NewAlertHandler(repo)

	body := `{
		"status": "resolved",
		"alerts": [
			{
				"status": "resolved",
				"labels": {
					"alertname": "HighTemperature",
					"asset_id": "motor-101",
					"severity": "critical"
				},
				"annotations": {
					"alert_name": "High Temperature",
					"description": "Asset motor-101 temperature is back to normal"
				}
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/alerts/webhook", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.AlertmanagerWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if len(repo.resolvedAlerts) != 1 {
		t.Fatalf("expected 1 resolved alert, got %d", len(repo.resolvedAlerts))
	}

	alert := repo.resolvedAlerts[0]

	if alert.AssetID != "motor-101" {
		t.Fatalf("expected asset motor-101, got %s", alert.AssetID)
	}

	if alert.Name != "High Temperature" {
		t.Fatalf("expected High Temperature alert, got %s", alert.Name)
	}

	if alert.Status != "RESOLVED" {
		t.Fatalf("expected status RESOLVED, got %s", alert.Status)
	}
}
