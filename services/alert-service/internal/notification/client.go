package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	EventCriticalAlertCreated  = "CRITICAL_ALERT_CREATED"
	EventIncidentCreated       = "INCIDENT_CREATED"
	EventIncidentAssigned      = "INCIDENT_ASSIGNED"
	EventIncidentAcknowledged  = "INCIDENT_ACKNOWLEDGED"
	EventIncidentResolved      = "INCIDENT_RESOLVED"
	EventIncidentClosed        = "INCIDENT_CLOSED"
	EventSLAAckBreached        = "SLA_ACK_BREACHED"
	EventSLAResolutionBreached = "SLA_RESOLUTION_BREACHED"
	EventIncidentEscalated     = "INCIDENT_ESCALATED"
)

type SendRequest struct {
	EventType  string         `json:"event_type"`
	Subject    string         `json:"subject"`
	Message    string         `json:"message"`
	Severity   string         `json:"severity,omitempty"`
	AssetID    string         `json:"asset_id,omitempty"`
	AlertID    *int64         `json:"alert_id,omitempty"`
	IncidentID *int64         `json:"incident_id,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *Client) Send(ctx context.Context, req SendRequest) {
	if c == nil || c.baseURL == "" {
		return
	}

	payload, err := json.Marshal(req)
	if err != nil {
		log.Printf("failed to marshal notification request: %v", err)
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/notifications/send", bytes.NewReader(payload))
	if err != nil {
		log.Printf("failed to create notification request: %v", err)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		log.Printf("notification-service unavailable: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("notification-service returned error status %d for %s", resp.StatusCode, req.EventType)
	}
}

func IncidentID(id int64) *int64 {
	return &id
}

func AlertID(id int64) *int64 {
	return &id
}

func CriticalAlertMessage(alertName string, assetID string) string {
	return fmt.Sprintf("Critical alert %s created for asset %s", alertName, assetID)
}
