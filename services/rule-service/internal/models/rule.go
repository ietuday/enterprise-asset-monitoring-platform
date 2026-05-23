package models

import "time"

type RuleStatus string

const (
	RuleStatusDraft    RuleStatus = "draft"
	RuleStatusActive   RuleStatus = "active"
	RuleStatusDisabled RuleStatus = "disabled"
	RuleStatusArchived RuleStatus = "archived"
)

type Rule struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Metric    string     `json:"metric"`
	Operator  string     `json:"operator"`
	Threshold float64    `json:"threshold"`
	Value     string     `json:"value,omitempty"`
	Severity  string     `json:"severity"`
	Enabled   bool       `json:"enabled"`
	Status    RuleStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type RuleVersion struct {
	ID        int64      `json:"id"`
	RuleID    int64      `json:"rule_id"`
	Version   int        `json:"version"`
	Name      string     `json:"name"`
	Metric    string     `json:"metric"`
	Operator  string     `json:"operator"`
	Threshold float64    `json:"threshold"`
	Value     string     `json:"value,omitempty"`
	Severity  string     `json:"severity"`
	Enabled   bool       `json:"enabled"`
	Status    RuleStatus `json:"status"`
	CreatedBy string     `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
}

func (s RuleStatus) IsValid() bool {
	switch s {
	case RuleStatusDraft, RuleStatusActive, RuleStatusDisabled, RuleStatusArchived:
		return true
	default:
		return false
	}
}
