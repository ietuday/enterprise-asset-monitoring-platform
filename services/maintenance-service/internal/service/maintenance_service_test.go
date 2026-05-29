package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"maintenance-service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestNormalizeCreateRequestDefaultsPriority(t *testing.T) {
	if NewMaintenanceService(nil) == nil {
		t.Fatalf("expected constructor to return a service")
	}

	req := normalizeCreateRequest(models.TaskCreateRequest{})
	if req.Priority != models.PriorityMedium {
		t.Fatalf("expected default priority %q, got %q", models.PriorityMedium, req.Priority)
	}

	req = normalizeCreateRequest(models.TaskCreateRequest{Priority: models.PriorityHigh})
	if req.Priority != models.PriorityHigh {
		t.Fatalf("expected explicit priority to be preserved")
	}
}

func TestBuildListTasksQuery(t *testing.T) {
	tests := []struct {
		name          string
		filters       models.TaskFilters
		wantFragments []string
		wantArgs      []any
	}{
		{
			name:          "no filters",
			filters:       models.TaskFilters{},
			wantFragments: []string{"WHERE 1 = 1", "ORDER BY due_date ASC"},
		},
		{
			name:          "status priority asset",
			filters:       models.TaskFilters{Status: models.StatusScheduled, Priority: models.PriorityHigh, AssetID: "motor-101"},
			wantFragments: []string{"status = $1", "asset_id = $2", "priority = $3"},
			wantArgs:      []any{models.StatusScheduled, "motor-101", models.PriorityHigh},
		},
		{
			name:          "overdue status",
			filters:       models.TaskFilters{Status: models.StatusOverdue},
			wantFragments: []string{"due_date < NOW()", "status NOT IN ('completed', 'cancelled')"},
		},
		{
			name:          "overdue flag",
			filters:       models.TaskFilters{Overdue: true},
			wantFragments: []string{"due_date < NOW()", "status NOT IN ('completed', 'cancelled')"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args := buildListTasksQuery(tt.filters)
			for _, fragment := range tt.wantFragments {
				if !strings.Contains(query, fragment) {
					t.Fatalf("expected query to contain %q, got %s", fragment, query)
				}
			}
			if len(args) != len(tt.wantArgs) {
				t.Fatalf("expected %d args, got %d: %+v", len(tt.wantArgs), len(args), args)
			}
			for i := range tt.wantArgs {
				if args[i] != tt.wantArgs[i] {
					t.Fatalf("arg %d: expected %v, got %v", i, tt.wantArgs[i], args[i])
				}
			}
		})
	}
}

func TestApplyTaskUpdate(t *testing.T) {
	now := time.Now().UTC()
	later := now.Add(24 * time.Hour)
	task := models.MaintenanceTask{
		AssetID:         "motor-101",
		Title:           "Inspect motor",
		Description:     "old",
		MaintenanceType: "inspection",
		Priority:        models.PriorityMedium,
		Status:          models.StatusScheduled,
		ScheduledDate:   now,
		DueDate:         later,
		AssignedTo:      "old@example.com",
		CreatedBy:       "admin@example.com",
	}

	title := "Inspect pump"
	description := "new"
	priority := models.PriorityCritical
	status := models.StatusInProgress
	assignedTo := "operator@example.com"
	next, err := applyTaskUpdate(task, models.TaskUpdateRequest{
		Title:       &title,
		Description: &description,
		Priority:    &priority,
		Status:      &status,
		AssignedTo:  &assignedTo,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next.Title != title || next.Description != description || next.Priority != priority || next.Status != status || next.AssignedTo != assignedTo {
		t.Fatalf("update was not applied: %+v", next)
	}
	if next.AssetID != task.AssetID || next.DueDate != task.DueDate {
		t.Fatalf("unchanged fields should be preserved: %+v", next)
	}
}

func TestApplyTaskUpdateRejectsCompletedAndInvalidDates(t *testing.T) {
	now := time.Now().UTC()
	task := models.MaintenanceTask{
		Status:        models.StatusCompleted,
		ScheduledDate: now,
		DueDate:       now.Add(time.Hour),
	}

	if _, err := applyTaskUpdate(task, models.TaskUpdateRequest{}); !errors.Is(err, ErrCompletedTaskLocked) {
		t.Fatalf("expected completed task lock, got %v", err)
	}

	task.Status = models.StatusScheduled
	dueBeforeSchedule := now.Add(-time.Hour)
	if _, err := applyTaskUpdate(task, models.TaskUpdateRequest{DueDate: &dueBeforeSchedule}); !errors.Is(err, ErrInvalidTaskDates) {
		t.Fatalf("expected invalid date error, got %v", err)
	}
}

func TestItoa(t *testing.T) {
	if itoa(0) != "0" || itoa(42) != "42" {
		t.Fatalf("itoa returned unexpected values")
	}
}

func TestCreateTaskService(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name    string
		req     models.TaskCreateRequest
		db      *fakeDB
		wantErr error
	}{
		{
			name: "success explicit priority",
			req:  validCreateRequest(now, models.PriorityHigh),
			db: &fakeDB{beginTx: &fakeTx{
				queryRow: fakeTaskRow(sampleServiceTask(now, models.PriorityHigh, models.StatusScheduled), nil),
			}},
		},
		{
			name: "success default priority",
			req:  validCreateRequest(now, ""),
			db: &fakeDB{beginTx: &fakeTx{
				queryRow: fakeTaskRow(sampleServiceTask(now, models.PriorityMedium, models.StatusScheduled), nil),
			}},
		},
		{name: "empty title", req: models.TaskCreateRequest{Title: " ", MaintenanceType: "inspection", Priority: models.PriorityMedium, ScheduledDate: now, DueDate: now.Add(time.Hour)}, db: &fakeDB{}, wantErr: ErrTitleRequired},
		{name: "empty type", req: models.TaskCreateRequest{Title: "Inspect", Priority: models.PriorityMedium, ScheduledDate: now, DueDate: now.Add(time.Hour)}, db: &fakeDB{}, wantErr: ErrMaintenanceTypeRequired},
		{name: "invalid priority", req: validCreateRequest(now, "urgent"), db: &fakeDB{}, wantErr: ErrInvalidPriority},
		{name: "bad dates", req: models.TaskCreateRequest{Title: "Inspect", MaintenanceType: "inspection", Priority: models.PriorityMedium, ScheduledDate: now, DueDate: now.Add(-time.Hour)}, db: &fakeDB{}, wantErr: ErrInvalidTaskDates},
		{name: "begin error", req: validCreateRequest(now, models.PriorityMedium), db: &fakeDB{beginErr: errors.New("begin failed")}, wantErr: errors.New("begin failed")},
		{name: "insert error", req: validCreateRequest(now, models.PriorityMedium), db: &fakeDB{beginTx: &fakeTx{queryRow: fakeRow{err: errors.New("insert failed")}}}, wantErr: errors.New("insert failed")},
		{name: "history error", req: validCreateRequest(now, models.PriorityMedium), db: &fakeDB{beginTx: &fakeTx{queryRow: fakeTaskRow(sampleServiceTask(now, models.PriorityMedium, models.StatusScheduled), nil), execErr: errors.New("history failed")}}, wantErr: errors.New("history failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MaintenanceService{db: tt.db}
			task, err := service.CreateTask(context.Background(), tt.req)
			if tt.wantErr != nil {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if task.Status != models.StatusScheduled {
				t.Fatalf("expected scheduled task, got %+v", task)
			}
			if tt.db.beginTx.committed != 1 {
				t.Fatalf("expected transaction commit")
			}
		})
	}
}

func TestListAndGetTaskService(t *testing.T) {
	now := time.Now().UTC()
	task := sampleServiceTask(now, models.PriorityHigh, models.StatusScheduled)

	t.Run("list success", func(t *testing.T) {
		db := &fakeDB{queryRows: fakeRowsFromTasks([]models.MaintenanceTask{task}, nil)}
		service := &MaintenanceService{db: db}
		tasks, err := service.ListTasks(context.Background(), models.TaskFilters{Status: models.StatusScheduled, Priority: models.PriorityHigh, AssetID: "asset-1", Overdue: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tasks) != 1 || tasks[0].ID != task.ID {
			t.Fatalf("unexpected tasks: %+v", tasks)
		}
		if !strings.Contains(db.lastQuery, "asset_id") || len(db.lastArgs) != 3 {
			t.Fatalf("expected filtered query, got %s args=%+v", db.lastQuery, db.lastArgs)
		}
	})

	t.Run("list query error", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{queryErr: errors.New("query failed")}}
		if _, err := service.ListTasks(context.Background(), models.TaskFilters{}); err == nil || !strings.Contains(err.Error(), "query failed") {
			t.Fatalf("expected query error, got %v", err)
		}
	})

	t.Run("list scan error", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{queryRows: &fakeRows{rows: [][]any{{"bad-id"}}}}}
		if _, err := service.ListTasks(context.Background(), models.TaskFilters{}); err == nil {
			t.Fatalf("expected scan error")
		}
	})

	t.Run("get success", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{row: fakeTaskRow(task, nil)}}
		got, err := service.GetTask(context.Background(), "1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != task.ID {
			t.Fatalf("unexpected task: %+v", got)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{row: fakeRow{err: pgx.ErrNoRows}}}
		if _, err := service.GetTask(context.Background(), "missing"); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("expected pgx.ErrNoRows, got %v", err)
		}
	})
}

func TestUpdateTaskService(t *testing.T) {
	now := time.Now().UTC()
	current := sampleServiceTask(now, models.PriorityMedium, models.StatusScheduled)
	updated := sampleServiceTask(now, models.PriorityCritical, models.StatusInProgress)
	updated.Title = "Updated"
	title := updated.Title
	description := "fresh details"
	maintenanceType := "calibration"
	priority := models.PriorityCritical
	status := models.StatusInProgress
	assignedTo := "operator@example.com"
	scheduled := now.Add(2 * time.Hour)
	due := now.Add(4 * time.Hour)

	t.Run("success", func(t *testing.T) {
		db := &fakeDB{
			row: fakeTaskRow(current, nil),
			beginTx: &fakeTx{
				queryRow: fakeTaskRow(updated, nil),
			},
		}
		service := &MaintenanceService{db: db}
		task, err := service.UpdateTask(context.Background(), "1", models.TaskUpdateRequest{
			Title:           &title,
			Description:     &description,
			MaintenanceType: &maintenanceType,
			Priority:        &priority,
			Status:          &status,
			ScheduledDate:   &scheduled,
			DueDate:         &due,
			AssignedTo:      &assignedTo,
		}, "admin@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Title != updated.Title || db.beginTx.committed != 1 {
			t.Fatalf("unexpected update result: %+v committed=%d", task, db.beginTx.committed)
		}
	})

	for _, tt := range []struct {
		name string
		req  models.TaskUpdateRequest
		err  error
	}{
		{name: "empty title", req: models.TaskUpdateRequest{Title: ptr(" ")}, err: ErrTitleRequired},
		{name: "empty type", req: models.TaskUpdateRequest{MaintenanceType: ptr("")}, err: ErrMaintenanceTypeRequired},
		{name: "invalid priority", req: models.TaskUpdateRequest{Priority: ptr("urgent")}, err: ErrInvalidPriority},
		{name: "invalid status", req: models.TaskUpdateRequest{Status: ptr("done")}, err: ErrInvalidStatus},
		{name: "completed status", req: models.TaskUpdateRequest{Status: ptr(models.StatusCompleted)}, err: ErrCompletedTaskLocked},
		{name: "bad dates", req: models.TaskUpdateRequest{ScheduledDate: &due, DueDate: &scheduled}, err: ErrInvalidTaskDates},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &MaintenanceService{db: &fakeDB{row: fakeTaskRow(current, nil)}}
			if _, err := service.UpdateTask(context.Background(), "1", tt.req, "actor"); !errors.Is(err, tt.err) {
				t.Fatalf("expected %v, got %v", tt.err, err)
			}
		})
	}

	t.Run("not found", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{row: fakeRow{err: pgx.ErrNoRows}}}
		if _, err := service.UpdateTask(context.Background(), "1", models.TaskUpdateRequest{}, "actor"); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("expected no rows, got %v", err)
		}
	})

	t.Run("update error", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{row: fakeTaskRow(current, nil), beginTx: &fakeTx{queryRow: fakeRow{err: errors.New("update failed")}}}}
		if _, err := service.UpdateTask(context.Background(), "1", models.TaskUpdateRequest{}, "actor"); err == nil || !strings.Contains(err.Error(), "update failed") {
			t.Fatalf("expected update error, got %v", err)
		}
	})

	t.Run("history error", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{row: fakeTaskRow(current, nil), beginTx: &fakeTx{queryRow: fakeTaskRow(updated, nil), execErr: errors.New("history failed")}}}
		if _, err := service.UpdateTask(context.Background(), "1", models.TaskUpdateRequest{}, "actor"); err == nil || !strings.Contains(err.Error(), "history failed") {
			t.Fatalf("expected history error, got %v", err)
		}
	})
}

func TestStatusCompleteCancelAndHistoryService(t *testing.T) {
	now := time.Now().UTC()
	current := sampleServiceTask(now, models.PriorityMedium, models.StatusScheduled)
	inProgress := sampleServiceTask(now, models.PriorityMedium, models.StatusInProgress)
	completed := sampleServiceTask(now, models.PriorityMedium, models.StatusCompleted)
	cancelled := sampleServiceTask(now, models.PriorityMedium, models.StatusCancelled)

	t.Run("change status success", func(t *testing.T) {
		db := &fakeDB{row: fakeTaskRow(current, nil), beginTx: &fakeTx{queryRow: fakeTaskRow(inProgress, nil)}}
		service := &MaintenanceService{db: db}
		task, err := service.ChangeStatus(context.Background(), "1", models.StatusChangeRequest{Status: models.StatusInProgress})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Status != models.StatusInProgress || db.beginTx.committed != 1 {
			t.Fatalf("unexpected task: %+v", task)
		}
	})

	t.Run("change invalid status", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{}}
		if _, err := service.ChangeStatus(context.Background(), "1", models.StatusChangeRequest{Status: "done"}); !errors.Is(err, ErrInvalidStatus) {
			t.Fatalf("expected invalid status, got %v", err)
		}
	})

	t.Run("change update error", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{row: fakeTaskRow(current, nil), beginTx: &fakeTx{queryRow: fakeRow{err: errors.New("update failed")}}}}
		if _, err := service.ChangeStatus(context.Background(), "1", models.StatusChangeRequest{Status: models.StatusInProgress}); err == nil || !strings.Contains(err.Error(), "update failed") {
			t.Fatalf("expected update error, got %v", err)
		}
	})

	t.Run("change history error", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{row: fakeTaskRow(current, nil), beginTx: &fakeTx{queryRow: fakeTaskRow(inProgress, nil), execErr: errors.New("history failed")}}}
		if _, err := service.ChangeStatus(context.Background(), "1", models.StatusChangeRequest{Status: models.StatusInProgress}); err == nil || !strings.Contains(err.Error(), "history failed") {
			t.Fatalf("expected history error, got %v", err)
		}
	})

	for _, tt := range []struct {
		name   string
		run    func(*MaintenanceService) (*models.MaintenanceTask, error)
		result models.MaintenanceTask
	}{
		{name: "complete success", run: func(s *MaintenanceService) (*models.MaintenanceTask, error) {
			return s.CompleteTask(context.Background(), "1", models.CompletionRequest{Comment: "done"})
		}, result: completed},
		{name: "cancel success", run: func(s *MaintenanceService) (*models.MaintenanceTask, error) {
			return s.CancelTask(context.Background(), "1", models.CompletionRequest{Comment: "cancel"})
		}, result: cancelled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := &fakeDB{row: fakeTaskRow(current, nil), beginTx: &fakeTx{queryRow: fakeTaskRow(tt.result, nil)}}
			service := &MaintenanceService{db: db}
			task, err := tt.run(service)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if task.Status != tt.result.Status || db.beginTx.committed != 1 {
				t.Fatalf("unexpected result: %+v", task)
			}
		})
	}

	t.Run("finalize not found", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{row: fakeRow{err: pgx.ErrNoRows}}}
		if _, err := service.CompleteTask(context.Background(), "1", models.CompletionRequest{}); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("expected no rows, got %v", err)
		}
	})

	t.Run("finalize update error", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{row: fakeTaskRow(current, nil), beginTx: &fakeTx{queryRow: fakeRow{err: errors.New("update failed")}}}}
		if _, err := service.CancelTask(context.Background(), "1", models.CompletionRequest{}); err == nil || !strings.Contains(err.Error(), "update failed") {
			t.Fatalf("expected update error, got %v", err)
		}
	})

	t.Run("list history success", func(t *testing.T) {
		history := []models.MaintenanceHistory{{ID: 1, TaskID: 1, Action: models.ActionTaskCreated, CreatedAt: now}}
		service := &MaintenanceService{db: &fakeDB{queryRows: fakeRowsFromHistory(history, nil)}}
		got, err := service.ListHistory(context.Background(), "1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0].Action != models.ActionTaskCreated {
			t.Fatalf("unexpected history: %+v", got)
		}
	})

	t.Run("list history query error", func(t *testing.T) {
		service := &MaintenanceService{db: &fakeDB{queryErr: errors.New("history query failed")}}
		if _, err := service.ListHistory(context.Background(), "1"); err == nil || !strings.Contains(err.Error(), "history query failed") {
			t.Fatalf("expected query error, got %v", err)
		}
	})
}

func validCreateRequest(now time.Time, priority string) models.TaskCreateRequest {
	return models.TaskCreateRequest{
		AssetID:         "asset-1",
		Title:           "Inspect motor",
		MaintenanceType: "inspection",
		Priority:        priority,
		ScheduledDate:   now,
		DueDate:         now.Add(2 * time.Hour),
		AssignedTo:      "operator@example.com",
		CreatedBy:       "admin@example.com",
	}
}

func sampleServiceTask(now time.Time, priority string, status string) models.MaintenanceTask {
	return models.MaintenanceTask{
		ID:              1,
		AssetID:         "asset-1",
		Title:           "Inspect motor",
		Description:     "check vibration",
		MaintenanceType: "inspection",
		Priority:        priority,
		Status:          status,
		ScheduledDate:   now,
		DueDate:         now.Add(2 * time.Hour),
		AssignedTo:      "operator@example.com",
		CreatedBy:       "admin@example.com",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func ptr(value string) *string {
	return &value
}

type fakeDB struct {
	beginTx   *fakeTx
	beginErr  error
	queryRows pgx.Rows
	queryErr  error
	row       pgx.Row
	lastQuery string
	lastArgs  []any
}

func (f *fakeDB) Begin(_ context.Context) (pgx.Tx, error) {
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	if f.beginTx == nil {
		f.beginTx = &fakeTx{}
	}
	return f.beginTx, nil
}

func (f *fakeDB) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	f.lastQuery = sql
	f.lastArgs = args
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return f.queryRows, nil
}

func (f *fakeDB) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	f.lastQuery = sql
	f.lastArgs = args
	if f.row == nil {
		return fakeRow{err: pgx.ErrNoRows}
	}
	return f.row
}

type fakeTx struct {
	queryRow  pgx.Row
	execErr   error
	committed int
	rolled    int
}

func (f *fakeTx) Begin(context.Context) (pgx.Tx, error) { return nil, errors.New("not implemented") }
func (f *fakeTx) Commit(context.Context) error {
	f.committed++
	return nil
}
func (f *fakeTx) Rollback(context.Context) error {
	f.rolled++
	return nil
}
func (f *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (f *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (f *fakeTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (f *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, f.execErr
}
func (f *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row {
	if f.queryRow == nil {
		return fakeRow{err: pgx.ErrNoRows}
	}
	return f.queryRow
}
func (f *fakeTx) Conn() *pgx.Conn { return nil }

type fakeRow struct {
	values []any
	err    error
}

func (f fakeRow) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(f.values) < len(dest) {
		return errors.New("not enough scan values")
	}
	for i := range dest {
		switch target := dest[i].(type) {
		case *int64:
			value, ok := f.values[i].(int64)
			if !ok {
				return errors.New("invalid int64 scan value")
			}
			*target = value
		case *string:
			value, ok := f.values[i].(string)
			if !ok {
				return errors.New("invalid string scan value")
			}
			*target = value
		case *time.Time:
			value, ok := f.values[i].(time.Time)
			if !ok {
				return errors.New("invalid time scan value")
			}
			*target = value
		case **time.Time:
			value, _ := f.values[i].(*time.Time)
			*target = value
		default:
			return errors.New("unsupported scan target")
		}
	}
	return nil
}

type fakeRows struct {
	rows   [][]any
	index  int
	err    error
	closed bool
}

func (f *fakeRows) Close()                                       { f.closed = true }
func (f *fakeRows) Err() error                                   { return f.err }
func (f *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (f *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (f *fakeRows) Next() bool {
	if f.index >= len(f.rows) {
		return false
	}
	f.index++
	return true
}
func (f *fakeRows) Scan(dest ...any) error {
	return fakeRow{values: f.rows[f.index-1]}.Scan(dest...)
}
func (f *fakeRows) Values() ([]any, error) { return f.rows[f.index-1], nil }
func (f *fakeRows) RawValues() [][]byte    { return nil }
func (f *fakeRows) Conn() *pgx.Conn        { return nil }

func fakeTaskRow(task models.MaintenanceTask, err error) fakeRow {
	return fakeRow{values: taskValues(task), err: err}
}

func fakeRowsFromTasks(tasks []models.MaintenanceTask, err error) *fakeRows {
	rows := make([][]any, 0, len(tasks))
	for _, task := range tasks {
		rows = append(rows, taskValues(task))
	}
	return &fakeRows{rows: rows, err: err}
}

func fakeRowsFromHistory(items []models.MaintenanceHistory, err error) *fakeRows {
	rows := make([][]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, []any{item.ID, item.TaskID, item.Action, item.OldStatus, item.NewStatus, item.Comment, item.PerformedBy, item.CreatedAt})
	}
	return &fakeRows{rows: rows, err: err}
}

func taskValues(task models.MaintenanceTask) []any {
	return []any{
		task.ID,
		task.AssetID,
		task.Title,
		task.Description,
		task.MaintenanceType,
		task.Priority,
		task.Status,
		task.ScheduledDate,
		task.DueDate,
		task.CompletedAt,
		task.AssignedTo,
		task.CreatedBy,
		task.CreatedAt,
		task.UpdatedAt,
	}
}
