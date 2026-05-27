package models

import "time"

const (
	IncidentStatusOpen         = "OPEN"
	IncidentStatusAssigned     = "ASSIGNED"
	IncidentStatusAcknowledged = "ACKNOWLEDGED"
	IncidentStatusResolved     = "RESOLVED"
	IncidentStatusClosed       = "CLOSED"
)

const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityMedium   = "MEDIUM"
	SeverityLow      = "LOW"
)

type Alert struct {
	ID         int64      `json:"id"`
	AssetID    string     `json:"assetId"`
	Name       string     `json:"name"`
	Severity   string     `json:"severity"`
	Status     string     `json:"status"`
	Message    string     `json:"message"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type Incident struct {
	ID             int64      `json:"id"`
	AlertID        *int64     `json:"alert_id,omitempty"`
	AssetID        string     `json:"asset_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Severity       string     `json:"severity"`
	Status         string     `json:"status"`
	AssignedTo     *string    `json:"assigned_to,omitempty"`
	ResolutionNote *string    `json:"resolution_note,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
}

type IncidentHistory struct {
	ID         int64     `json:"id"`
	IncidentID int64     `json:"incident_id"`
	Action     string    `json:"action"`
	OldStatus  *string   `json:"old_status,omitempty"`
	NewStatus  string    `json:"new_status"`
	Actor      string    `json:"actor"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

type IncidentFilters struct {
	Status     string
	Severity   string
	AssignedTo string
}
