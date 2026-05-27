package models

import (
	"encoding/json"
	"time"
)

const (
	ChannelTypeEmail   = "EMAIL"
	ChannelTypeWebhook = "WEBHOOK"
)

const (
	NotificationStatusPending = "PENDING"
	NotificationStatusSent    = "SENT"
	NotificationStatusFailed  = "FAILED"
)

const (
	EventCriticalAlertCreated = "CRITICAL_ALERT_CREATED"
	EventIncidentCreated      = "INCIDENT_CREATED"
	EventIncidentAssigned     = "INCIDENT_ASSIGNED"
	EventIncidentAcknowledged = "INCIDENT_ACKNOWLEDGED"
	EventIncidentResolved     = "INCIDENT_RESOLVED"
	EventIncidentClosed       = "INCIDENT_CLOSED"
	EventTestNotification     = "TEST_NOTIFICATION"
)

type NotificationChannel struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Target    string    `json:"target"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NotificationHistory struct {
	ID           int64           `json:"id"`
	EventType    string          `json:"event_type"`
	ChannelID    *int64          `json:"channel_id,omitempty"`
	ChannelName  string          `json:"channel_name"`
	ChannelType  string          `json:"channel_type"`
	Recipient    string          `json:"recipient"`
	Subject      string          `json:"subject"`
	Message      string          `json:"message"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	Status       string          `json:"status"`
	ErrorMessage *string         `json:"error_message,omitempty"`
	RetryCount   int             `json:"retry_count"`
	CreatedAt    time.Time       `json:"created_at"`
	SentAt       *time.Time      `json:"sent_at,omitempty"`
}

type SendNotificationRequest struct {
	EventType  string          `json:"event_type"`
	Subject    string          `json:"subject"`
	Message    string          `json:"message"`
	Severity   string          `json:"severity,omitempty"`
	AssetID    string          `json:"asset_id,omitempty"`
	AlertID    *int64          `json:"alert_id,omitempty"`
	IncidentID *int64          `json:"incident_id,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

type TestNotificationRequest struct {
	ChannelID int64  `json:"channel_id"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
}

type HistoryFilters struct {
	Status      string
	ChannelType string
	EventType   string
}

type SendResult struct {
	ChannelID    int64  `json:"channel_id"`
	ChannelName  string `json:"channel_name"`
	ChannelType  string `json:"channel_type"`
	Recipient    string `json:"recipient"`
	HistoryID    int64  `json:"history_id"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type SendSummary struct {
	Total   int          `json:"total"`
	Sent    int          `json:"sent"`
	Failed  int          `json:"failed"`
	Results []SendResult `json:"results"`
}
