package rules

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

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

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient() *Client {
	baseURL := os.Getenv("RULE_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5004"
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) GetEnabledRules() ([]Rule, error) {
	url := fmt.Sprintf("%s/rules/enabled", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch enabled rules: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rule service returned status %d", resp.StatusCode)
	}

	var rules []Rule
	if err := json.NewDecoder(resp.Body).Decode(&rules); err != nil {
		return nil, fmt.Errorf("failed to decode rules response: %w", err)
	}

	return rules, nil
}
