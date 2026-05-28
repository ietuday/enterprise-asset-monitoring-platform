package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"notification-service/internal/models"
	"notification-service/internal/service"

	"github.com/go-chi/chi/v5"
)

type NotificationService interface {
	CreateChannel(rctx context.Context, channel *models.NotificationChannel) error
	ListChannels(rctx context.Context) ([]models.NotificationChannel, error)
	GetChannelByID(rctx context.Context, id string) (*models.NotificationChannel, error)
	UpdateChannel(rctx context.Context, channel *models.NotificationChannel) error
	DeleteChannel(rctx context.Context, id string) error
	SetChannelEnabled(rctx context.Context, id string, enabled bool) (*models.NotificationChannel, error)
	Send(rctx context.Context, req models.SendNotificationRequest) (models.SendSummary, error)
	Test(rctx context.Context, req models.TestNotificationRequest) (models.SendResult, error)
	ListHistory(rctx context.Context, filters models.HistoryFilters) ([]models.NotificationHistory, error)
	GetHistoryByID(rctx context.Context, id string) (*models.NotificationHistory, error)
	Retry(rctx context.Context, id string) (models.SendResult, error)
}

type Handler struct {
	service NotificationService
}

type channelRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Target  string `json:"target"`
	Enabled *bool  `json:"enabled"`
}

func NewHandler(service NotificationService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "notification-service",
	})
}

func (h *Handler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	var req channelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	channel := models.NotificationChannel{
		Name:    req.Name,
		Type:    req.Type,
		Target:  req.Target,
		Enabled: enabled,
	}

	if err := h.service.CreateChannel(r.Context(), &channel); err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, channel)
}

func (h *Handler) ListChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := h.service.ListChannels(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, channels)
}

func (h *Handler) GetChannelByID(w http.ResponseWriter, r *http.Request) {
	channel, err := h.service.GetChannelByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, channel)
}

func (h *Handler) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	var req channelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	channel := models.NotificationChannel{
		ID:      id,
		Name:    req.Name,
		Type:    req.Type,
		Target:  req.Target,
		Enabled: req.Enabled != nil && *req.Enabled,
	}
	if err := h.service.UpdateChannel(r.Context(), &channel); err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, channel)
}

func (h *Handler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteChannel(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) EnableChannel(w http.ResponseWriter, r *http.Request) {
	h.setChannelEnabled(w, r, true)
}

func (h *Handler) DisableChannel(w http.ResponseWriter, r *http.Request) {
	h.setChannelEnabled(w, r, false)
}

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	var req models.SendNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	summary, err := h.service.Send(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, summary)
}

func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
	var req models.TestNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.Test(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, result)
}

func (h *Handler) ListHistory(w http.ResponseWriter, r *http.Request) {
	filters := models.HistoryFilters{
		Status:      r.URL.Query().Get("status"),
		ChannelType: r.URL.Query().Get("channel_type"),
		EventType:   r.URL.Query().Get("event_type"),
	}

	history, err := h.service.ListHistory(r.Context(), filters)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, history)
}

func (h *Handler) GetHistoryByID(w http.ResponseWriter, r *http.Request) {
	history, err := h.service.GetHistoryByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, history)
}

func (h *Handler) Retry(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Retry(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, result)
}

func (h *Handler) setChannelEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	channel, err := h.service.SetChannelEnabled(r.Context(), chi.URLParam(r, "id"), enabled)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, channel)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrUnsupportedStatus):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "resource not found")
	case errors.Is(err, service.ErrRetryNotAllowed):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{
		"error": strings.TrimPrefix(message, service.ErrInvalidInput.Error()+": "),
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
