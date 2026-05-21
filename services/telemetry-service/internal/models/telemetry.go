package models

import "time"

type Telemetry struct {
	ID          int64     `json:"id"`
	AssetID     string    `json:"assetId"`
	Temperature float64   `json:"temperature"`
	CPU         float64   `json:"cpu"`
	Memory      float64   `json:"memory"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}
