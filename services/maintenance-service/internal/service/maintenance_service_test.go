package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"maintenance-service/internal/models"
)

func TestNormalizeCreateRequestDefaultsPriority(t *testing.T) {
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
