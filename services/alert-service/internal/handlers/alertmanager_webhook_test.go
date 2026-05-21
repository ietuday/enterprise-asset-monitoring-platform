package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"alert-service/internal/models"
)

type fakeAlertRepository struct {
	createdAlerts  []models.Alert
	resolvedAlerts []models.Alert
}

func (f *fakeAlertRepository) Create(ctx context.Context, alert *models.Alert) error {
	alert.ID = int64(len(f.createdAlerts) + 1)
	f.createdAlerts = append(f.createdAlerts, *alert)
	return nil
}

func (f *fakeAlertRepository) List(ctx context.Context) ([]models.Alert, error) {
	return nil, nil
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
