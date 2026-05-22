package models

import "time"

type Rule struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Metric    string    `json:"metric"`
	Operator  string    `json:"operator"`
	Threshold float64   `json:"threshold"`
	Value     string    `json:"value,omitempty"`
	Severity  string    `json:"severity"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}