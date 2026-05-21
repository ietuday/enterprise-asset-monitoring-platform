package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	rulesclient "telemetry-service/internal/clients/rules"
	"telemetry-service/internal/metrics"
	"telemetry-service/internal/models"
	"telemetry-service/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type TelemetryHandler struct {
	repo                  *repository.TelemetryRepository
	alertServiceURL       string
	directAlertingEnabled bool
	ruleClient            *rulesclient.Client
}

type resolveActiveAlertRequest struct {
	AssetID string `json:"assetId"`
	Name    string `json:"name"`
}

type alertRequest struct {
	AssetID  string `json:"assetId"`
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

func NewTelemetryHandler(repo *repository.TelemetryRepository) *TelemetryHandler {
	alertServiceURL := os.Getenv("ALERT_SERVICE_URL")
	if alertServiceURL == "" {
		alertServiceURL = "http://localhost:5003"
	}

	directAlertingEnabled := os.Getenv("ENABLE_DIRECT_ALERTING") == "true"

	return &TelemetryHandler{
		repo:                  repo,
		alertServiceURL:       alertServiceURL,
		directAlertingEnabled: directAlertingEnabled,
		ruleClient:            rulesclient.NewClient(),
	}
}

func (h *TelemetryHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "telemetry-service",
		"status":  "healthy",
	})
}

func (h *TelemetryHandler) CreateTelemetry(w http.ResponseWriter, r *http.Request) {
	var telemetry models.Telemetry

	if err := json.NewDecoder(r.Body).Decode(&telemetry); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if telemetry.AssetID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "assetId is required",
		})
		return
	}

	if telemetry.Status == "" {
		telemetry.Status = "UNKNOWN"
	}

	if err := h.repo.Create(r.Context(), &telemetry); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Business metric: total telemetry received.
	metrics.TelemetryReceivedTotal.WithLabelValues(
		telemetry.AssetID,
		telemetry.Status,
	).Inc()

	// Latest telemetry gauges.
	// Prometheus rules use these metrics for alert evaluation.
	metrics.AssetTemperatureCelsius.WithLabelValues(
		telemetry.AssetID,
	).Set(telemetry.Temperature)

	metrics.AssetCPUUsagePercent.WithLabelValues(
		telemetry.AssetID,
	).Set(telemetry.CPU)

	metrics.AssetMemoryUsagePercent.WithLabelValues(
		telemetry.AssetID,
	).Set(telemetry.Memory)

	// Dynamic rules are evaluated here for debug visibility.
	// Actual Prometheus/Alertmanager alerting is handled by generated Prometheus rules.
	h.evaluateDynamicRules(telemetry)

	// Legacy/direct alerting path.
	// This is only enabled when ENABLE_DIRECT_ALERTING=true.
	// When ENABLE_DIRECT_ALERTING=false, Prometheus + Alertmanager handle alert creation/resolution.
	if h.directAlertingEnabled {
		h.evaluateRulesAndCreateAlerts(telemetry)
	}

	writeJSON(w, http.StatusCreated, telemetry)
}

func (h *TelemetryHandler) GetLatestTelemetry(w http.ResponseWriter, r *http.Request) {
	assetID := chi.URLParam(r, "assetId")

	telemetry, err := h.repo.GetLatestByAssetID(r.Context(), assetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "telemetry not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, telemetry)
}

// evaluateRulesAndCreateAlerts is the legacy/direct alerting path.
// It is executed only when ENABLE_DIRECT_ALERTING=true.
// In Prometheus/Alertmanager mode, alerting is handled through metrics,
// Prometheus alert rules, and Alertmanager webhook delivery.
func (h *TelemetryHandler) evaluateRulesAndCreateAlerts(telemetry models.Telemetry) {
	if telemetry.Temperature > 80 {
		h.createAlert(alertRequest{
			AssetID:  telemetry.AssetID,
			Name:     "High Temperature",
			Severity: "CRITICAL",
			Status:   "OPEN",
			Message:  "Temperature is above threshold",
		})
	} else {
		h.resolveActiveAlert(telemetry.AssetID, "High Temperature")
	}

	if telemetry.CPU > 90 {
		h.createAlert(alertRequest{
			AssetID:  telemetry.AssetID,
			Name:     "High CPU Usage",
			Severity: "HIGH",
			Status:   "OPEN",
			Message:  "CPU usage is above threshold",
		})
	} else {
		h.resolveActiveAlert(telemetry.AssetID, "High CPU Usage")
	}

	if telemetry.Memory > 90 {
		h.createAlert(alertRequest{
			AssetID:  telemetry.AssetID,
			Name:     "High Memory Usage",
			Severity: "HIGH",
			Status:   "OPEN",
			Message:  "Memory usage is above threshold",
		})
	} else {
		h.resolveActiveAlert(telemetry.AssetID, "High Memory Usage")
	}

	if telemetry.Status == "DOWN" {
		h.createAlert(alertRequest{
			AssetID:  telemetry.AssetID,
			Name:     "Device Down",
			Severity: "CRITICAL",
			Status:   "OPEN",
			Message:  "Device status is DOWN",
		})
	} else {
		h.resolveActiveAlert(telemetry.AssetID, "Device Down")
	}
}

func (h *TelemetryHandler) createAlert(alert alertRequest) {
	body, err := json.Marshal(alert)
	if err != nil {
		log.Printf("failed to marshal alert request: %v", err)
		return
	}

	url := h.alertServiceURL + "/alerts"

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("failed to call alert service: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("alert service returned non-success status: %d", resp.StatusCode)
		return
	}

	log.Printf("alert created successfully for asset=%s name=%s", alert.AssetID, alert.Name)
}

func (h *TelemetryHandler) resolveActiveAlert(assetID string, name string) {
	req := resolveActiveAlertRequest{
		AssetID: assetID,
		Name:    name,
	}

	body, err := json.Marshal(req)
	if err != nil {
		log.Printf("failed to marshal resolve alert request: %v", err)
		return
	}

	url := h.alertServiceURL + "/alerts/resolve-active"

	request, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("failed to create resolve alert request: %v", err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(request)
	if err != nil {
		log.Printf("failed to call alert resolve API: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return
	}

	if resp.StatusCode >= 300 {
		log.Printf("alert resolve API returned non-success status: %d", resp.StatusCode)
		return
	}

	log.Printf("active alert resolved successfully for asset=%s name=%s", assetID, name)
}

func evaluateNumericRule(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}

func telemetryMetricValue(telemetry models.Telemetry, metric string) (float64, bool) {
	switch metric {
	case "temperature":
		return telemetry.Temperature, true
	case "cpu":
		return telemetry.CPU, true
	case "memory":
		return telemetry.Memory, true
	default:
		return 0, false
	}
}

// evaluateDynamicRules fetches enabled rules from Rule Service and evaluates them
// against the current telemetry payload for logging/debug visibility.
//
// Note:
// This does not create alerts directly.
// Dynamic Prometheus alerting is handled by rule-service generating Prometheus
// rules from PostgreSQL and reloading Prometheus.
func (h *TelemetryHandler) evaluateDynamicRules(telemetry models.Telemetry) {
	rules, err := h.ruleClient.GetEnabledRules()
	if err != nil {
		log.Printf("failed to fetch dynamic rules: %v", err)
		return
	}

	for _, rule := range rules {
		value, ok := telemetryMetricValue(telemetry, rule.Metric)
		if !ok {
			log.Printf("unsupported dynamic rule metric: %s", rule.Metric)
			continue
		}

		matched := evaluateNumericRule(value, rule.Operator, rule.Threshold)

		log.Printf(
			"dynamic rule evaluated asset_id=%s rule=%q metric=%s value=%f operator=%s threshold=%f matched=%t",
			telemetry.AssetID,
			rule.Name,
			rule.Metric,
			value,
			rule.Operator,
			rule.Threshold,
			matched,
		)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}