package service

import (
	"context"
	"errors"
	"strings"

	"maintenance-service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCompletedTaskLocked = errors.New("completed maintenance tasks cannot be modified")
var ErrInvalidTaskDates = errors.New("due_date cannot be before scheduled_date")

type MaintenanceService struct {
	db *pgxpool.Pool
}

func NewMaintenanceService(db *pgxpool.Pool) *MaintenanceService {
	return &MaintenanceService{db: db}
}

func (s *MaintenanceService) CreateTask(ctx context.Context, req models.TaskCreateRequest) (*models.MaintenanceTask, error) {
	req = normalizeCreateRequest(req)

	task := &models.MaintenanceTask{}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO maintenance_tasks (
			asset_id, title, description, maintenance_type, priority, status,
			scheduled_date, due_date, assigned_to, created_by
		)
		VALUES ($1, $2, $3, $4, $5, 'scheduled', $6, $7, $8, $9)
		RETURNING id, asset_id, title, description, maintenance_type, priority, status,
			scheduled_date, due_date, completed_at, assigned_to, created_by, created_at, updated_at;
	`,
		req.AssetID,
		req.Title,
		req.Description,
		req.MaintenanceType,
		req.Priority,
		req.ScheduledDate,
		req.DueDate,
		req.AssignedTo,
		req.CreatedBy,
	)
	if err := scanTask(row, task); err != nil {
		return nil, err
	}

	if err := insertHistory(ctx, tx, task.ID, models.ActionTaskCreated, "", task.Status, "Task created", req.CreatedBy); err != nil {
		return nil, err
	}

	return task, tx.Commit(ctx)
}

func (s *MaintenanceService) ListTasks(ctx context.Context, filters models.TaskFilters) ([]models.MaintenanceTask, error) {
	query, args := buildListTasksQuery(filters)
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (s *MaintenanceService) GetTask(ctx context.Context, id string) (*models.MaintenanceTask, error) {
	task := &models.MaintenanceTask{}
	if err := scanTask(s.db.QueryRow(ctx, selectTaskSQL("id = $1"), id), task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *MaintenanceService) UpdateTask(ctx context.Context, id string, req models.TaskUpdateRequest, actor string) (*models.MaintenanceTask, error) {
	current, err := s.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}

	next, err := applyTaskUpdate(*current, req)
	if err != nil {
		return nil, err
	}

	task := &models.MaintenanceTask{}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		UPDATE maintenance_tasks
		SET asset_id = $2, title = $3, description = $4, maintenance_type = $5,
			priority = $6, status = $7, scheduled_date = $8, due_date = $9,
			assigned_to = $10, created_by = $11, updated_at = NOW()
		WHERE id = $1
		RETURNING id, asset_id, title, description, maintenance_type, priority, status,
			scheduled_date, due_date, completed_at, assigned_to, created_by, created_at, updated_at;
	`,
		id,
		next.AssetID,
		next.Title,
		next.Description,
		next.MaintenanceType,
		next.Priority,
		next.Status,
		next.ScheduledDate,
		next.DueDate,
		next.AssignedTo,
		next.CreatedBy,
	)
	if err := scanTask(row, task); err != nil {
		return nil, err
	}

	if err := insertHistory(ctx, tx, task.ID, models.ActionTaskUpdated, current.Status, task.Status, "Task updated", actor); err != nil {
		return nil, err
	}

	return task, tx.Commit(ctx)
}

func buildListTasksQuery(filters models.TaskFilters) (string, []any) {
	conditions := []string{"1 = 1"}
	args := []any{}

	if filters.Status != "" {
		if filters.Status == models.StatusOverdue {
			conditions = append(conditions, "due_date < NOW() AND status NOT IN ('completed', 'cancelled')")
		} else {
			args = append(args, filters.Status)
			conditions = append(conditions, "status = $"+itoa(len(args)))
		}
	}
	if filters.AssetID != "" {
		args = append(args, filters.AssetID)
		conditions = append(conditions, "asset_id = $"+itoa(len(args)))
	}
	if filters.Priority != "" {
		args = append(args, filters.Priority)
		conditions = append(conditions, "priority = $"+itoa(len(args)))
	}
	if filters.Overdue {
		conditions = append(conditions, "due_date < NOW() AND status NOT IN ('completed', 'cancelled')")
	}

	query := `
		SELECT id, asset_id, title, description, maintenance_type, priority,
			CASE
				WHEN due_date < NOW() AND status NOT IN ('completed', 'cancelled') THEN 'overdue'
				ELSE status
			END AS status,
			scheduled_date, due_date, completed_at, assigned_to, created_by, created_at, updated_at
		FROM maintenance_tasks
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY due_date ASC, created_at DESC;
	`

	return query, args
}

func normalizeCreateRequest(req models.TaskCreateRequest) models.TaskCreateRequest {
	if req.Priority == "" {
		req.Priority = models.PriorityMedium
	}
	return req
}

func applyTaskUpdate(current models.MaintenanceTask, req models.TaskUpdateRequest) (models.MaintenanceTask, error) {
	if current.Status == models.StatusCompleted {
		return models.MaintenanceTask{}, ErrCompletedTaskLocked
	}

	next := current
	if req.AssetID != nil {
		next.AssetID = *req.AssetID
	}
	if req.Title != nil {
		next.Title = *req.Title
	}
	if req.Description != nil {
		next.Description = *req.Description
	}
	if req.MaintenanceType != nil {
		next.MaintenanceType = *req.MaintenanceType
	}
	if req.Priority != nil {
		next.Priority = *req.Priority
	}
	if req.Status != nil {
		next.Status = *req.Status
	}
	if req.ScheduledDate != nil {
		next.ScheduledDate = *req.ScheduledDate
	}
	if req.DueDate != nil {
		next.DueDate = *req.DueDate
	}
	if req.AssignedTo != nil {
		next.AssignedTo = *req.AssignedTo
	}
	if req.CreatedBy != nil {
		next.CreatedBy = *req.CreatedBy
	}
	if next.DueDate.Before(next.ScheduledDate) {
		return models.MaintenanceTask{}, ErrInvalidTaskDates
	}

	return next, nil
}

func (s *MaintenanceService) ChangeStatus(ctx context.Context, id string, req models.StatusChangeRequest) (*models.MaintenanceTask, error) {
	current, err := s.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}

	task := &models.MaintenanceTask{}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		UPDATE maintenance_tasks
		SET status = $2, completed_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE completed_at END, updated_at = NOW()
		WHERE id = $1
		RETURNING id, asset_id, title, description, maintenance_type, priority, status,
			scheduled_date, due_date, completed_at, assigned_to, created_by, created_at, updated_at;
	`, id, req.Status)
	if err := scanTask(row, task); err != nil {
		return nil, err
	}

	if err := insertHistory(ctx, tx, task.ID, models.ActionStatusChanged, current.Status, task.Status, req.Comment, req.PerformedBy); err != nil {
		return nil, err
	}

	return task, tx.Commit(ctx)
}

func (s *MaintenanceService) CompleteTask(ctx context.Context, id string, req models.CompletionRequest) (*models.MaintenanceTask, error) {
	return s.finalizeTask(ctx, id, models.StatusCompleted, models.ActionTaskCompleted, req.Comment, req.PerformedBy)
}

func (s *MaintenanceService) CancelTask(ctx context.Context, id string, req models.CompletionRequest) (*models.MaintenanceTask, error) {
	return s.finalizeTask(ctx, id, models.StatusCancelled, models.ActionTaskCancelled, req.Comment, req.PerformedBy)
}

func (s *MaintenanceService) ListHistory(ctx context.Context, id string) ([]models.MaintenanceHistory, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, task_id, action, old_status, new_status, comment, performed_by, created_at
		FROM maintenance_history
		WHERE task_id = $1
		ORDER BY created_at ASC, id ASC;
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.MaintenanceHistory, 0)
	for rows.Next() {
		var item models.MaintenanceHistory
		if err := rows.Scan(
			&item.ID,
			&item.TaskID,
			&item.Action,
			&item.OldStatus,
			&item.NewStatus,
			&item.Comment,
			&item.PerformedBy,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *MaintenanceService) finalizeTask(ctx context.Context, id string, status string, action string, comment string, actor string) (*models.MaintenanceTask, error) {
	current, err := s.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}

	task := &models.MaintenanceTask{}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	completedSQL := "completed_at"
	if status == models.StatusCompleted {
		completedSQL = "NOW()"
	}

	row := tx.QueryRow(ctx, `
		UPDATE maintenance_tasks
		SET status = $2, completed_at = `+completedSQL+`, updated_at = NOW()
		WHERE id = $1
		RETURNING id, asset_id, title, description, maintenance_type, priority, status,
			scheduled_date, due_date, completed_at, assigned_to, created_by, created_at, updated_at;
	`, id, status)
	if err := scanTask(row, task); err != nil {
		return nil, err
	}

	if err := insertHistory(ctx, tx, task.ID, action, current.Status, task.Status, comment, actor); err != nil {
		return nil, err
	}

	return task, tx.Commit(ctx)
}

type queryRow interface {
	Scan(dest ...any) error
}

func selectTaskSQL(where string) string {
	return `
		SELECT id, asset_id, title, description, maintenance_type, priority,
			CASE
				WHEN due_date < NOW() AND status NOT IN ('completed', 'cancelled') THEN 'overdue'
				ELSE status
			END AS status,
			scheduled_date, due_date, completed_at, assigned_to, created_by, created_at, updated_at
		FROM maintenance_tasks
		WHERE ` + where + `;
	`
}

func scanTask(row queryRow, task *models.MaintenanceTask) error {
	return row.Scan(
		&task.ID,
		&task.AssetID,
		&task.Title,
		&task.Description,
		&task.MaintenanceType,
		&task.Priority,
		&task.Status,
		&task.ScheduledDate,
		&task.DueDate,
		&task.CompletedAt,
		&task.AssignedTo,
		&task.CreatedBy,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
}

func scanTasks(rows pgx.Rows) ([]models.MaintenanceTask, error) {
	tasks := make([]models.MaintenanceTask, 0)
	for rows.Next() {
		var task models.MaintenanceTask
		if err := scanTask(rows, &task); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

func insertHistory(ctx context.Context, tx pgx.Tx, taskID int64, action string, oldStatus string, newStatus string, comment string, actor string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO maintenance_history (task_id, action, old_status, new_status, comment, performed_by)
		VALUES ($1, $2, $3, $4, $5, $6);
	`, taskID, action, oldStatus, newStatus, comment, actor)
	return err
}

func itoa(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}

	out := make([]byte, 0, 8)
	for value > 0 {
		out = append([]byte{digits[value%10]}, out...)
		value = value / 10
	}
	return string(out)
}
