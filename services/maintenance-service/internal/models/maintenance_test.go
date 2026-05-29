package models

import "testing"

func TestIsValidStatus(t *testing.T) {
	validStatuses := []string{
		StatusScheduled,
		StatusInProgress,
		StatusCompleted,
		StatusOverdue,
		StatusCancelled,
	}

	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			if !IsValidStatus(status) {
				t.Fatalf("expected %q to be valid", status)
			}
		})
	}

	if IsValidStatus("unknown") {
		t.Fatal("expected unknown status to be invalid")
	}
}

func TestIsValidPriority(t *testing.T) {
	validPriorities := []string{
		PriorityLow,
		PriorityMedium,
		PriorityHigh,
		PriorityCritical,
	}

	for _, priority := range validPriorities {
		t.Run(priority, func(t *testing.T) {
			if !IsValidPriority(priority) {
				t.Fatalf("expected %q to be valid", priority)
			}
		})
	}

	if IsValidPriority("unknown") {
		t.Fatal("expected unknown priority to be invalid")
	}
}
