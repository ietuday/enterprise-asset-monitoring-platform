package models

import (
	"encoding/json"
	"time"
)

type RuleAuditLog struct {
	ID        int64           `json:"id"`
	RuleID    *int64          `json:"rule_id"`
	Action    string          `json:"action"`
	RuleName  string          `json:"rule_name"`
	OldValue  json.RawMessage `json:"old_value,omitempty"`
	NewValue  json.RawMessage `json:"new_value,omitempty"`
	ChangedBy string          `json:"changed_by"`
	CreatedAt time.Time       `json:"created_at"`
}
