package models

import "time"

const (
	StatusScheduled  = "scheduled"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusOverdue    = "overdue"
	StatusCancelled  = "cancelled"
)

const (
	PriorityLow      = "low"
	PriorityMedium   = "medium"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)

const (
	ActionTaskCreated   = "TASK_CREATED"
	ActionTaskUpdated   = "TASK_UPDATED"
	ActionStatusChanged = "STATUS_CHANGED"
	ActionTaskCompleted = "TASK_COMPLETED"
	ActionTaskCancelled = "TASK_CANCELLED"
)

type MaintenanceTask struct {
	ID              int64      `json:"id"`
	AssetID         string     `json:"asset_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	MaintenanceType string     `json:"maintenance_type"`
	Priority        string     `json:"priority"`
	Status          string     `json:"status"`
	ScheduledDate   time.Time  `json:"scheduled_date"`
	DueDate         time.Time  `json:"due_date"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	AssignedTo      string     `json:"assigned_to"`
	CreatedBy       string     `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type TaskFilters struct {
	Status   string
	AssetID  string
	Priority string
	Overdue  bool
}

type TaskCreateRequest struct {
	AssetID         string    `json:"asset_id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	MaintenanceType string    `json:"maintenance_type"`
	Priority        string    `json:"priority"`
	ScheduledDate   time.Time `json:"scheduled_date"`
	DueDate         time.Time `json:"due_date"`
	AssignedTo      string    `json:"assigned_to"`
	CreatedBy       string    `json:"created_by"`
}

type TaskUpdateRequest struct {
	AssetID         *string    `json:"asset_id"`
	Title           *string    `json:"title"`
	Description     *string    `json:"description"`
	MaintenanceType *string    `json:"maintenance_type"`
	Priority        *string    `json:"priority"`
	Status          *string    `json:"status"`
	ScheduledDate   *time.Time `json:"scheduled_date"`
	DueDate         *time.Time `json:"due_date"`
	AssignedTo      *string    `json:"assigned_to"`
	CreatedBy       *string    `json:"created_by"`
}

type StatusChangeRequest struct {
	Status      string `json:"status"`
	Comment     string `json:"comment"`
	PerformedBy string `json:"performed_by"`
}

type CompletionRequest struct {
	Comment     string `json:"comment"`
	PerformedBy string `json:"performed_by"`
}

type MaintenanceHistory struct {
	ID          int64     `json:"id"`
	TaskID      int64     `json:"task_id"`
	Action      string    `json:"action"`
	OldStatus   string    `json:"old_status,omitempty"`
	NewStatus   string    `json:"new_status,omitempty"`
	Comment     string    `json:"comment"`
	PerformedBy string    `json:"performed_by"`
	CreatedAt   time.Time `json:"created_at"`
}

func IsValidStatus(status string) bool {
	switch status {
	case StatusScheduled, StatusInProgress, StatusCompleted, StatusOverdue, StatusCancelled:
		return true
	default:
		return false
	}
}

func IsValidPriority(priority string) bool {
	switch priority {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return true
	default:
		return false
	}
}
