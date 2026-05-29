package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"maintenance-service/internal/models"
	"maintenance-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

const (
	errInvalidStatus   = "invalid status"
	errInvalidPriority = "invalid priority"
)

type maintenanceService interface {
	ListTasks(ctx context.Context, filters models.TaskFilters) ([]models.MaintenanceTask, error)
	CreateTask(ctx context.Context, req models.TaskCreateRequest) (*models.MaintenanceTask, error)
	GetTask(ctx context.Context, id string) (*models.MaintenanceTask, error)
	UpdateTask(ctx context.Context, id string, req models.TaskUpdateRequest, actor string) (*models.MaintenanceTask, error)
	ChangeStatus(ctx context.Context, id string, req models.StatusChangeRequest) (*models.MaintenanceTask, error)
	CompleteTask(ctx context.Context, id string, req models.CompletionRequest) (*models.MaintenanceTask, error)
	CancelTask(ctx context.Context, id string, req models.CompletionRequest) (*models.MaintenanceTask, error)
	ListHistory(ctx context.Context, id string) ([]models.MaintenanceHistory, error)
}

type MaintenanceHandler struct {
	service maintenanceService
}

func NewMaintenanceHandler(service maintenanceService) *MaintenanceHandler {
	return &MaintenanceHandler{service: service}
}

func (h *MaintenanceHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "maintenance-service",
		"status":  "healthy",
	})
}

func (h *MaintenanceHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filters := models.TaskFilters{
		Status:   strings.TrimSpace(query.Get("status")),
		AssetID:  strings.TrimSpace(query.Get("asset_id")),
		Priority: strings.TrimSpace(query.Get("priority")),
		Overdue:  strings.EqualFold(query.Get("overdue"), "true"),
	}

	if filters.Status != "" && !models.IsValidStatus(filters.Status) {
		writeError(w, http.StatusBadRequest, errInvalidStatus)
		return
	}
	if filters.Priority != "" && !models.IsValidPriority(filters.Priority) {
		writeError(w, http.StatusBadRequest, errInvalidPriority)
		return
	}

	tasks, err := h.service.ListTasks(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

func (h *MaintenanceHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req models.TaskCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if validation := validateCreate(req); validation != "" {
		writeError(w, http.StatusBadRequest, validation)
		return
	}

	task, err := h.service.CreateTask(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *MaintenanceHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	task, err := h.service.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleTaskError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *MaintenanceHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	var req models.TaskUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if validation := validateUpdate(req); validation != "" {
		writeError(w, http.StatusBadRequest, validation)
		return
	}

	actor := r.Header.Get("x-user-email")
	if actor == "" {
		actor = r.Header.Get("x-user-id")
	}

	task, err := h.service.UpdateTask(r.Context(), chi.URLParam(r, "id"), req, actor)
	if err != nil {
		handleTaskError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *MaintenanceHandler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	var req models.StatusChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	if !models.IsValidStatus(req.Status) {
		writeError(w, http.StatusBadRequest, errInvalidStatus)
		return
	}

	task, err := h.service.ChangeStatus(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		handleTaskError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *MaintenanceHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeCompletion(w, r)
	if !ok {
		return
	}

	task, err := h.service.CompleteTask(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		handleTaskError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *MaintenanceHandler) CancelTask(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeCompletion(w, r)
	if !ok {
		return
	}

	task, err := h.service.CancelTask(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		handleTaskError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *MaintenanceHandler) ListAssetTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.service.ListTasks(r.Context(), models.TaskFilters{
		AssetID: chi.URLParam(r, "assetId"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

func (h *MaintenanceHandler) ListOverdueTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.service.ListTasks(r.Context(), models.TaskFilters{Overdue: true})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

func (h *MaintenanceHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListHistory(r.Context(), chi.URLParam(r, "taskId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func validateCreate(req models.TaskCreateRequest) string {
	if strings.TrimSpace(req.AssetID) == "" {
		return "asset_id is required"
	}
	if strings.TrimSpace(req.Title) == "" {
		return "title is required"
	}
	if strings.TrimSpace(req.MaintenanceType) == "" {
		return "maintenance_type is required"
	}
	if req.ScheduledDate.IsZero() || req.DueDate.IsZero() {
		return "scheduled_date and due_date are required"
	}
	if req.DueDate.Before(req.ScheduledDate) {
		return "due_date cannot be before scheduled_date"
	}
	if req.Priority == "" {
		req.Priority = models.PriorityMedium
	}
	if !models.IsValidPriority(req.Priority) {
		return errInvalidPriority
	}
	return ""
}

func validateUpdate(req models.TaskUpdateRequest) string {
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return "title cannot be empty"
	}
	if req.MaintenanceType != nil && strings.TrimSpace(*req.MaintenanceType) == "" {
		return "maintenance_type cannot be empty"
	}
	if req.Priority != nil && !models.IsValidPriority(*req.Priority) {
		return errInvalidPriority
	}
	if req.Status != nil && !models.IsValidStatus(*req.Status) {
		return errInvalidStatus
	}
	if req.Status != nil && *req.Status == models.StatusCompleted {
		return "use the complete endpoint to complete maintenance tasks"
	}
	if req.ScheduledDate != nil && req.DueDate != nil && req.DueDate.Before(*req.ScheduledDate) {
		return "due_date cannot be before scheduled_date"
	}
	return ""
}

func decodeCompletion(w http.ResponseWriter, r *http.Request) (models.CompletionRequest, bool) {
	var req models.CompletionRequest
	if r.Body == nil {
		return req, true
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return req, false
	}
	return req, true
}

func handleTaskError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "maintenance task not found")
		return
	}
	if errors.Is(err, service.ErrCompletedTaskLocked) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, service.ErrInvalidTaskDates) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
