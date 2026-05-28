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

const (
	SLAStatusOnTrack            = "ON_TRACK"
	SLAStatusAckBreached        = "ACK_BREACHED"
	SLAStatusResolutionBreached = "RESOLUTION_BREACHED"
	SLAStatusEscalated          = "ESCALATED"
	SLAStatusCompleted          = "COMPLETED"
	SLAStatusNoPolicy           = "NO_POLICY"
)

const (
	EscalationActionSLAAckBreached        = "SLA_ACK_BREACHED"
	EscalationActionSLAResolutionBreached = "SLA_RESOLUTION_BREACHED"
	EscalationActionIncidentEscalated     = "INCIDENT_ESCALATED"
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

type SLAPolicy struct {
	ID                       int64     `json:"id"`
	Severity                 string    `json:"severity"`
	AcknowledgeWithinMinutes int       `json:"acknowledge_within_minutes"`
	ResolveWithinMinutes     int       `json:"resolve_within_minutes"`
	EscalationTarget         string    `json:"escalation_target"`
	Enabled                  bool      `json:"enabled"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type IncidentSLATracking struct {
	ID               int64      `json:"id"`
	IncidentID       int64      `json:"incident_id"`
	Severity         string     `json:"severity"`
	Status           string     `json:"status"`
	AcknowledgeDueAt *time.Time `json:"acknowledge_due_at,omitempty"`
	ResolveDueAt     *time.Time `json:"resolve_due_at,omitempty"`
	AcknowledgedAt   *time.Time `json:"acknowledged_at,omitempty"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
	EscalatedAt      *time.Time `json:"escalated_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type EscalationHistory struct {
	ID         int64     `json:"id"`
	IncidentID int64     `json:"incident_id"`
	Action     string    `json:"action"`
	Reason     string    `json:"reason"`
	Target     string    `json:"target"`
	Actor      string    `json:"actor"`
	CreatedAt  time.Time `json:"created_at"`
}

type SLABreachFilters struct {
	Status     string
	Severity   string
	IncidentID string
}

type ManualEscalationRequest struct {
	Reason string `json:"reason"`
	Target string `json:"target"`
	Actor  string `json:"actor"`
}
