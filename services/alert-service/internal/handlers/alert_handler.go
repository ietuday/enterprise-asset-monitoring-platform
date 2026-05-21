package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"alert-service/internal/metrics"
	"alert-service/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type AlertRepository interface {
	Create(ctx context.Context, alert *models.Alert) error
	List(ctx context.Context) ([]models.Alert, error)
	GetByID(ctx context.Context, id string) (*models.Alert, error)
	Acknowledge(ctx context.Context, id string) (*models.Alert, error)
	Resolve(ctx context.Context, id string) (*models.Alert, error)
	ResolveActiveByAssetAndName(ctx context.Context, assetID string, name string) (*models.Alert, error)
}

type AlertHandler struct {
	repo AlertRepository
}

type resolveActiveAlertRequest struct {
	AssetID string `json:"assetId"`
	Name    string `json:"name"`
}

type alertmanagerWebhookRequest struct {
	Status string                   `json:"status"`
	Alerts []alertmanagerAlertEntry `json:"alerts"`
}

type alertmanagerAlertEntry struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

func NewAlertHandler(repo AlertRepository) *AlertHandler {
	return &AlertHandler{repo: repo}
}

func (h *AlertHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "alert-service",
		"status":  "healthy",
	})
}

func (h *AlertHandler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	var alert models.Alert

	if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if alert.AssetID == "" || alert.Name == "" || alert.Severity == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "assetId, name and severity are required",
		})
		return
	}

	if alert.Status == "" {
		alert.Status = "OPEN"
	}

	if alert.Message == "" {
		alert.Message = alert.Name
	}

	if err := h.repo.Create(r.Context(), &alert); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	metrics.AlertsCreatedTotal.WithLabelValues(
		alert.AssetID,
		alert.Name,
		alert.Severity,
	).Inc()

	writeJSON(w, http.StatusCreated, alert)
}

func (h *AlertHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.repo.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, alerts)
}

func (h *AlertHandler) GetAlertByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	alert, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "alert not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, alert)
}

func (h *AlertHandler) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	alert, err := h.repo.Acknowledge(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "alert not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, alert)
}

func (h *AlertHandler) ResolveAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	alert, err := h.repo.Resolve(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "alert not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	metrics.AlertsResolvedTotal.WithLabelValues(
		alert.AssetID,
		alert.Name,
		alert.Severity,
	).Inc()

	writeJSON(w, http.StatusOK, alert)
}

func (h *AlertHandler) ResolveActiveAlert(w http.ResponseWriter, r *http.Request) {
	var req resolveActiveAlertRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if req.AssetID == "" || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "assetId and name are required",
		})
		return
	}

	alert, err := h.repo.ResolveActiveByAssetAndName(r.Context(), req.AssetID, req.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"message": "no active alert found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	metrics.AlertsResolvedTotal.WithLabelValues(
		alert.AssetID,
		alert.Name,
		alert.Severity,
	).Inc()

	writeJSON(w, http.StatusOK, alert)
}

func (h *AlertHandler) AlertmanagerWebhook(w http.ResponseWriter, r *http.Request) {
	var req alertmanagerWebhookRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid alertmanager webhook body",
		})
		return
	}

	processed := 0

	for _, item := range req.Alerts {
		assetID := item.Labels["asset_id"]
		if assetID == "" {
			assetID = item.Annotations["asset_id"]
		}

		alertName := item.Annotations["alert_name"]
		if alertName == "" {
			alertName = item.Labels["alertname"]
		}

		severity := item.Labels["severity"]
		if severity == "" {
			severity = "critical"
		}

		message := item.Annotations["description"]
		if message == "" {
			message = item.Annotations["summary"]
		}
		if message == "" {
			message = alertName
		}

		if assetID == "" || alertName == "" {
			continue
		}

		switch item.Status {
		case "firing":
			alert := models.Alert{
				AssetID:  assetID,
				Name:     alertName,
				Severity: strings.ToUpper(severity),
				Status:   "OPEN",
				Message:  message,
			}

			if err := h.repo.Create(r.Context(), &alert); err != nil {
				log.Printf("failed to create alert from alertmanager webhook: %v", err)
				continue
			}

			metrics.AlertsCreatedTotal.WithLabelValues(
				alert.AssetID,
				alert.Name,
				alert.Severity,
			).Inc()

			processed++

		case "resolved":
			alert, err := h.repo.ResolveActiveByAssetAndName(r.Context(), assetID, alertName)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}

				log.Printf("failed to resolve alert from alertmanager webhook: %v", err)
				continue
			}

			metrics.AlertsResolvedTotal.WithLabelValues(
				alert.AssetID,
				alert.Name,
				alert.Severity,
			).Inc()

			processed++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":   "alertmanager webhook processed",
		"processed": processed,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
