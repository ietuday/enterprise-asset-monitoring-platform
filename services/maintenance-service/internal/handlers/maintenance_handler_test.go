package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"maintenance-service/internal/models"

	"github.com/go-chi/chi/v5"
)

type fakeMaintenanceService struct {
	tasks          []models.MaintenanceTask
	history        []models.MaintenanceHistory
	err            error
	lastFilters    models.TaskFilters
	lastCreate     models.TaskCreateRequest
	lastStatus     models.StatusChangeRequest
	lastCompletion models.CompletionRequest
}

func (f *fakeMaintenanceService) ListTasks(_ context.Context, filters models.TaskFilters) ([]models.MaintenanceTask, error) {
	f.lastFilters = filters
	return f.tasks, f.err
}

func (f *fakeMaintenanceService) CreateTask(_ context.Context, req models.TaskCreateRequest) (*models.MaintenanceTask, error) {
	f.lastCreate = req
	if f.err != nil {
		return nil, f.err
	}
	task := sampleTask()
	task.Title = req.Title
	task.Priority = req.Priority
	if task.Priority == "" {
		task.Priority = models.PriorityMedium
	}
	return &task, nil
}

func (f *fakeMaintenanceService) GetTask(_ context.Context, _ string) (*models.MaintenanceTask, error) {
	if f.err != nil {
		return nil, f.err
	}
	task := sampleTask()
	return &task, nil
}

func (f *fakeMaintenanceService) UpdateTask(_ context.Context, _ string, _ models.TaskUpdateRequest, _ string) (*models.MaintenanceTask, error) {
	if f.err != nil {
		return nil, f.err
	}
	task := sampleTask()
	return &task, nil
}

func (f *fakeMaintenanceService) ChangeStatus(_ context.Context, _ string, req models.StatusChangeRequest) (*models.MaintenanceTask, error) {
	f.lastStatus = req
	if f.err != nil {
		return nil, f.err
	}
	task := sampleTask()
	task.Status = req.Status
	return &task, nil
}

func (f *fakeMaintenanceService) CompleteTask(_ context.Context, _ string, req models.CompletionRequest) (*models.MaintenanceTask, error) {
	f.lastCompletion = req
	if f.err != nil {
		return nil, f.err
	}
	task := sampleTask()
	now := time.Now()
	task.Status = models.StatusCompleted
	task.CompletedAt = &now
	return &task, nil
}

func (f *fakeMaintenanceService) CancelTask(_ context.Context, _ string, req models.CompletionRequest) (*models.MaintenanceTask, error) {
	f.lastCompletion = req
	if f.err != nil {
		return nil, f.err
	}
	task := sampleTask()
	task.Status = models.StatusCancelled
	return &task, nil
}

func (f *fakeMaintenanceService) ListHistory(_ context.Context, _ string) ([]models.MaintenanceHistory, error) {
	return f.history, f.err
}

func TestHealth(t *testing.T) {
	handler := NewMaintenanceHandler(&fakeMaintenanceService{})
	recorder := httptest.NewRecorder()

	handler.Health(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	assertJSONField(t, recorder.Body.Bytes(), "status", "healthy")
}

func TestListTasksRejectsInvalidFilters(t *testing.T) {
	tests := []struct {
		name string
		url  string
		err  string
	}{
		{name: "status", url: "/maintenance/tasks?status=bad", err: errInvalidStatus},
		{name: "priority", url: "/maintenance/tasks?priority=bad", err: errInvalidPriority},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewMaintenanceHandler(&fakeMaintenanceService{})
			recorder := httptest.NewRecorder()

			handler.ListTasks(recorder, httptest.NewRequest(http.MethodGet, tt.url, nil))

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", recorder.Code)
			}
			assertJSONField(t, recorder.Body.Bytes(), "error", tt.err)
		})
	}
}

func TestListTasksPassesFiltersToService(t *testing.T) {
	fake := &fakeMaintenanceService{tasks: []models.MaintenanceTask{sampleTask()}}
	handler := NewMaintenanceHandler(fake)
	recorder := httptest.NewRecorder()

	handler.ListTasks(recorder, httptest.NewRequest(http.MethodGet, "/maintenance/tasks?status=scheduled&priority=high&asset_id=motor-101&overdue=true", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if fake.lastFilters.Status != models.StatusScheduled || fake.lastFilters.Priority != models.PriorityHigh || fake.lastFilters.AssetID != "motor-101" || !fake.lastFilters.Overdue {
		t.Fatalf("unexpected filters: %+v", fake.lastFilters)
	}
}

func TestCreateTaskValidation(t *testing.T) {
	handler := NewMaintenanceHandler(&fakeMaintenanceService{})

	t.Run("invalid json", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.CreateTask(recorder, httptest.NewRequest(http.MethodPost, "/maintenance/tasks", bytes.NewBufferString("{")))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
		assertJSONField(t, recorder.Body.Bytes(), "error", "invalid request body")
	})

	t.Run("missing fields", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.CreateTask(recorder, httptest.NewRequest(http.MethodPost, "/maintenance/tasks", jsonBody(t, map[string]string{})))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
		assertJSONField(t, recorder.Body.Bytes(), "error", "asset_id is required")
	})
}

func TestCreateTaskSuccess(t *testing.T) {
	fake := &fakeMaintenanceService{}
	handler := NewMaintenanceHandler(fake)
	recorder := httptest.NewRecorder()
	scheduled := time.Now().Add(time.Hour).UTC()
	due := scheduled.Add(24 * time.Hour)

	handler.CreateTask(recorder, httptest.NewRequest(http.MethodPost, "/maintenance/tasks", jsonBody(t, map[string]any{
		"asset_id":         "motor-101",
		"title":            "Inspect motor",
		"maintenance_type": "inspection",
		"scheduled_date":   scheduled.Format(time.RFC3339),
		"due_date":         due.Format(time.RFC3339),
	})))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if fake.lastCreate.Title != "Inspect motor" {
		t.Fatalf("expected create request to reach service, got %+v", fake.lastCreate)
	}
	assertJSONField(t, recorder.Body.Bytes(), "status", models.StatusScheduled)
}

func TestChangeStatusRejectsInvalidStatus(t *testing.T) {
	handler := NewMaintenanceHandler(&fakeMaintenanceService{})
	recorder := httptest.NewRecorder()
	request := requestWithURLParam(http.MethodPatch, "/maintenance/tasks/1/status", "id", "1", jsonBody(t, map[string]string{"status": "bad"}))

	handler.ChangeStatus(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	assertJSONField(t, recorder.Body.Bytes(), "error", errInvalidStatus)
}

func TestCompleteCancelAndHistory(t *testing.T) {
	fake := &fakeMaintenanceService{
		history: []models.MaintenanceHistory{
			{ID: 1, TaskID: 1, Action: models.ActionTaskCreated},
			{ID: 2, TaskID: 1, Action: models.ActionTaskCompleted},
		},
	}
	handler := NewMaintenanceHandler(fake)

	t.Run("complete", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := requestWithURLParam(http.MethodPost, "/maintenance/tasks/1/complete", "id", "1", jsonBody(t, map[string]string{"comment": "done"}))
		handler.CompleteTask(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		assertJSONField(t, recorder.Body.Bytes(), "status", models.StatusCompleted)
	})

	t.Run("cancel", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := requestWithURLParam(http.MethodPost, "/maintenance/tasks/1/cancel", "id", "1", jsonBody(t, map[string]string{"comment": "cancel"}))
		handler.CancelTask(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		assertJSONField(t, recorder.Body.Bytes(), "status", models.StatusCancelled)
	})

	t.Run("history", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := requestWithURLParam(http.MethodGet, "/maintenance/history/1", "taskId", "1", nil)
		handler.ListHistory(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		var history []models.MaintenanceHistory
		if err := json.Unmarshal(recorder.Body.Bytes(), &history); err != nil {
			t.Fatalf("invalid response: %v", err)
		}
		if len(history) != 2 || history[1].Action != models.ActionTaskCompleted {
			t.Fatalf("unexpected history: %+v", history)
		}
	})
}

func TestHandlerMapsServiceErrors(t *testing.T) {
	handler := NewMaintenanceHandler(&fakeMaintenanceService{err: errors.New("boom")})
	recorder := httptest.NewRecorder()

	handler.ListTasks(recorder, httptest.NewRequest(http.MethodGet, "/maintenance/tasks", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
	assertJSONField(t, recorder.Body.Bytes(), "error", "boom")
}

func TestValidateCreateAndUpdate(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)
	empty := ""
	badPriority := "urgent"
	badStatus := "done"
	completed := models.StatusCompleted

	tests := []struct {
		name string
		err  string
		run  func() string
	}{
		{name: "empty title", err: "title is required", run: func() string {
			return validateCreate(models.TaskCreateRequest{AssetID: "a", MaintenanceType: "inspection", ScheduledDate: now, DueDate: later})
		}},
		{name: "empty type", err: "maintenance_type is required", run: func() string {
			return validateCreate(models.TaskCreateRequest{AssetID: "a", Title: "t", ScheduledDate: now, DueDate: later})
		}},
		{name: "bad date order", err: "due_date cannot be before scheduled_date", run: func() string {
			return validateCreate(models.TaskCreateRequest{AssetID: "a", Title: "t", MaintenanceType: "inspection", ScheduledDate: later, DueDate: now})
		}},
		{name: "bad priority", err: errInvalidPriority, run: func() string {
			return validateCreate(models.TaskCreateRequest{AssetID: "a", Title: "t", MaintenanceType: "inspection", Priority: badPriority, ScheduledDate: now, DueDate: later})
		}},
		{name: "update empty title", err: "title cannot be empty", run: func() string {
			return validateUpdate(models.TaskUpdateRequest{Title: &empty})
		}},
		{name: "update bad priority", err: errInvalidPriority, run: func() string {
			return validateUpdate(models.TaskUpdateRequest{Priority: &badPriority})
		}},
		{name: "update bad status", err: errInvalidStatus, run: func() string {
			return validateUpdate(models.TaskUpdateRequest{Status: &badStatus})
		}},
		{name: "update completed", err: "use the complete endpoint to complete maintenance tasks", run: func() string {
			return validateUpdate(models.TaskUpdateRequest{Status: &completed})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.run(); got != tt.err {
				t.Fatalf("expected %q, got %q", tt.err, got)
			}
		})
	}
}

func sampleTask() models.MaintenanceTask {
	now := time.Now().UTC()
	return models.MaintenanceTask{
		ID:              1,
		AssetID:         "motor-101",
		Title:           "Inspect motor",
		MaintenanceType: "inspection",
		Priority:        models.PriorityMedium,
		Status:          models.StatusScheduled,
		ScheduledDate:   now,
		DueDate:         now.Add(24 * time.Hour),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func jsonBody(t *testing.T, body any) *bytes.Reader {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return bytes.NewReader(payload)
}

func assertJSONField(t *testing.T, body []byte, field string, expected string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("invalid json response: %v body=%s", err, string(body))
	}
	if payload[field] != expected {
		t.Fatalf("expected %s=%q, got %v", field, expected, payload[field])
	}
}

func requestWithURLParam(method string, path string, key string, value string, body *bytes.Reader) *http.Request {
	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		requestBody = body
	}
	request := httptest.NewRequest(method, path, requestBody)
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add(key, value)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, ctx))
}
