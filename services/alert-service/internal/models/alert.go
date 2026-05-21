package models

import "time"

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
