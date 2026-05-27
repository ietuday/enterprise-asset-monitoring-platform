package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"alert-service/internal/metrics"
	"alert-service/internal/models"
	"alert-service/internal/notification"

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
	CreateIncident(ctx context.Context, incident *models.Incident, actor string, comment string) error
	ListIncidents(ctx context.Context, filters models.IncidentFilters) ([]models.Incident, error)
	GetIncidentByID(ctx context.Context, id string) (*models.Incident, error)
	AssignIncident(ctx context.Context, id string, assignedTo string, actor string, comment string) (*models.Incident, error)
	AcknowledgeIncident(ctx context.Context, id string, actor string, comment string) (*models.Incident, error)
	ResolveIncident(ctx context.Context, id string, actor string, resolutionNote string) (*models.Incident, error)
	CloseIncident(ctx context.Context, id string, actor string, comment string) (*models.Incident, error)
	GetIncidentHistory(ctx context.Context, incidentID string) ([]models.IncidentHistory, error)
	AddIncidentHistory(ctx context.Context, history *models.IncidentHistory) error
}

type AlertHandler struct {
	repo         AlertRepository
	notification NotificationClient
}

type NotificationClient interface {
	Send(ctx context.Context, req notification.SendRequest)
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

type createIncidentRequest struct {
	AlertID     *int64 `json:"alert_id"`
	AssetID     string `json:"asset_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Actor       string `json:"actor"`
	Comment     string `json:"comment"`
}

type assignIncidentRequest struct {
	AssignedTo string `json:"assigned_to"`
	Actor      string `json:"actor"`
	Comment    string `json:"comment"`
}

type incidentActionRequest struct {
	Actor          string `json:"actor"`
	Comment        string `json:"comment"`
	ResolutionNote string `json:"resolution_note"`
}

func NewAlertHandler(repo AlertRepository, notificationClients ...NotificationClient) *AlertHandler {
	var notificationClient NotificationClient
	if len(notificationClients) > 0 {
		notificationClient = notificationClients[0]
	}

	return &AlertHandler{repo: repo, notification: notificationClient}
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

	h.notifyCriticalAlertCreated(r.Context(), alert)

	if err := h.createIncidentForCriticalAlert(r.Context(), alert); err != nil {
		log.Printf("failed to auto-create incident for alert %d: %v", alert.ID, err)
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

			h.notifyCriticalAlertCreated(r.Context(), alert)

			if err := h.createIncidentForCriticalAlert(r.Context(), alert); err != nil {
				log.Printf("failed to auto-create incident for alert %d: %v", alert.ID, err)
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

func (h *AlertHandler) CreateIncident(w http.ResponseWriter, r *http.Request) {
	var req createIncidentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	req.Severity = strings.ToUpper(req.Severity)
	if req.AssetID == "" || req.Title == "" || req.Description == "" || req.Severity == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "asset_id, title, description and severity are required",
		})
		return
	}
	if !isValidSeverity(req.Severity) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "severity must be CRITICAL, HIGH, MEDIUM or LOW",
		})
		return
	}

	incident := models.Incident{
		AlertID:     req.AlertID,
		AssetID:     req.AssetID,
		Title:       req.Title,
		Description: req.Description,
		Severity:    req.Severity,
		Status:      models.IncidentStatusOpen,
	}

	if err := h.repo.CreateIncident(r.Context(), &incident, req.Actor, req.Comment); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	h.notifyIncident(r.Context(), notification.EventIncidentCreated, "Incident created", incident, "")

	writeJSON(w, http.StatusCreated, incident)
}

func (h *AlertHandler) ListIncidents(w http.ResponseWriter, r *http.Request) {
	filters := models.IncidentFilters{
		Status:     strings.ToUpper(r.URL.Query().Get("status")),
		Severity:   strings.ToUpper(r.URL.Query().Get("severity")),
		AssignedTo: r.URL.Query().Get("assigned_to"),
	}

	if filters.Status != "" && !isValidIncidentStatus(filters.Status) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid status filter",
		})
		return
	}

	if filters.Severity != "" && !isValidSeverity(filters.Severity) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid severity filter",
		})
		return
	}

	incidents, err := h.repo.ListIncidents(r.Context(), filters)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, incidents)
}

func (h *AlertHandler) GetIncidentByID(w http.ResponseWriter, r *http.Request) {
	incident, err := h.repo.GetIncidentByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "incident not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, incident)
}

func (h *AlertHandler) AssignIncident(w http.ResponseWriter, r *http.Request) {
	var req assignIncidentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if req.AssignedTo == "" || req.Actor == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "assigned_to and actor are required",
		})
		return
	}

	incident, err := h.repo.AssignIncident(r.Context(), chi.URLParam(r, "id"), req.AssignedTo, req.Actor, req.Comment)
	h.writeIncidentActionResponse(w, incident, err, func() {
		h.notifyIncident(r.Context(), notification.EventIncidentAssigned, "Incident assigned", *incident, req.AssignedTo)
	})
}

func (h *AlertHandler) AcknowledgeIncident(w http.ResponseWriter, r *http.Request) {
	var req incidentActionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if req.Actor == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "actor is required",
		})
		return
	}

	incident, err := h.repo.AcknowledgeIncident(r.Context(), chi.URLParam(r, "id"), req.Actor, req.Comment)
	h.writeIncidentActionResponse(w, incident, err, func() {
		h.notifyIncident(r.Context(), notification.EventIncidentAcknowledged, "Incident acknowledged", *incident, "")
	})
}

func (h *AlertHandler) ResolveIncident(w http.ResponseWriter, r *http.Request) {
	var req incidentActionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if req.Actor == "" || req.ResolutionNote == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "actor and resolution_note are required",
		})
		return
	}

	incident, err := h.repo.ResolveIncident(r.Context(), chi.URLParam(r, "id"), req.Actor, req.ResolutionNote)
	h.writeIncidentActionResponse(w, incident, err, func() {
		h.notifyIncident(r.Context(), notification.EventIncidentResolved, "Incident resolved", *incident, "")
	})
}

func (h *AlertHandler) CloseIncident(w http.ResponseWriter, r *http.Request) {
	var req incidentActionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if req.Actor == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "actor is required",
		})
		return
	}

	incident, err := h.repo.CloseIncident(r.Context(), chi.URLParam(r, "id"), req.Actor, req.Comment)
	h.writeIncidentActionResponse(w, incident, err, func() {
		h.notifyIncident(r.Context(), notification.EventIncidentClosed, "Incident closed", *incident, "")
	})
}

func (h *AlertHandler) GetIncidentHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := h.repo.GetIncidentByID(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "incident not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	history, err := h.repo.GetIncidentHistory(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, history)
}

func (h *AlertHandler) createIncidentForCriticalAlert(ctx context.Context, alert models.Alert) error {
	if strings.ToUpper(alert.Severity) != models.SeverityCritical {
		return nil
	}

	incident := models.Incident{
		AlertID:     &alert.ID,
		AssetID:     alert.AssetID,
		Title:       alert.Name,
		Description: alert.Message,
		Severity:    models.SeverityCritical,
		Status:      models.IncidentStatusOpen,
	}

	if err := h.repo.CreateIncident(ctx, &incident, "system", "Incident auto-created from critical alert"); err != nil {
		return err
	}

	h.notifyIncident(ctx, notification.EventIncidentCreated, "Incident created", incident, "")
	return nil
}

func isValidIncidentStatus(status string) bool {
	switch status {
	case models.IncidentStatusOpen,
		models.IncidentStatusAssigned,
		models.IncidentStatusAcknowledged,
		models.IncidentStatusResolved,
		models.IncidentStatusClosed:
		return true
	default:
		return false
	}
}

func isValidSeverity(severity string) bool {
	switch severity {
	case models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow:
		return true
	default:
		return false
	}
}

func (h *AlertHandler) writeIncidentActionResponse(w http.ResponseWriter, incident *models.Incident, err error, afterSuccess ...func()) {
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "incident not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	for _, hook := range afterSuccess {
		hook()
	}

	writeJSON(w, http.StatusOK, incident)
}

func (h *AlertHandler) notifyCriticalAlertCreated(ctx context.Context, alert models.Alert) {
	if strings.ToUpper(alert.Severity) != models.SeverityCritical || h.notification == nil {
		return
	}

	h.notification.Send(ctx, notification.SendRequest{
		EventType: notification.EventCriticalAlertCreated,
		Subject:   "Critical alert created",
		Message:   notification.CriticalAlertMessage(alert.Name, alert.AssetID),
		Severity:  alert.Severity,
		AssetID:   alert.AssetID,
		AlertID:   notification.AlertID(alert.ID),
		Payload: map[string]any{
			"alert_name": alert.Name,
			"status":     alert.Status,
		},
	})
}

func (h *AlertHandler) notifyIncident(ctx context.Context, eventType string, subject string, incident models.Incident, assignedTo string) {
	if h.notification == nil {
		return
	}

	message := ""
	switch eventType {
	case notification.EventIncidentCreated:
		message = "Incident #" + strconv.FormatInt(incident.ID, 10) + " created for asset " + incident.AssetID
	case notification.EventIncidentAssigned:
		message = "Incident #" + strconv.FormatInt(incident.ID, 10) + " assigned to " + assignedTo
	case notification.EventIncidentAcknowledged:
		message = "Incident #" + strconv.FormatInt(incident.ID, 10) + " acknowledged"
	case notification.EventIncidentResolved:
		message = "Incident #" + strconv.FormatInt(incident.ID, 10) + " resolved"
	case notification.EventIncidentClosed:
		message = "Incident #" + strconv.FormatInt(incident.ID, 10) + " closed"
	}

	h.notification.Send(ctx, notification.SendRequest{
		EventType:  eventType,
		Subject:    subject,
		Message:    message,
		Severity:   incident.Severity,
		AssetID:    incident.AssetID,
		AlertID:    incident.AlertID,
		IncidentID: notification.IncidentID(incident.ID),
		Payload: map[string]any{
			"status":      incident.Status,
			"assigned_to": incident.AssignedTo,
		},
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
