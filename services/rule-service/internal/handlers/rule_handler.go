package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"rule-service/internal/models"
	promrules "rule-service/internal/prometheus"
	"rule-service/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type RuleHandler struct {
	repo *repository.RuleRepository
}

func NewRuleHandler(repo *repository.RuleRepository) *RuleHandler {
	return &RuleHandler{repo: repo}
}

func (h *RuleHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "rule-service",
		"status":  "healthy",
	})
}

func (h *RuleHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var rule models.Rule

	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if rule.Status == "" {
		rule.Status = models.RuleStatusDraft
	}

	if !rule.Status.IsValid() {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid rule status",
		})
		return
	}

	if !isValidRule(rule) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "name, metric, operator and severity are required",
		})
		return
	}

	if !isSupportedMetric(rule.Metric) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "unsupported metric",
		})
		return
	}

	if !isSupportedOperator(rule.Operator) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "unsupported operator",
		})
		return
	}

	rule.Enabled = rule.Status == models.RuleStatusActive

	if err := h.repo.Create(r.Context(), &rule); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	ruleID := rule.ID

	if err := h.repo.CreateAuditLog(
		r.Context(),
		&ruleID,
		"RULE_CREATED",
		rule.Name,
		nil,
		rule,
		changedBy(r),
	); err != nil {
		log.Printf("failed to create rule audit log: %v", err)
	}

	h.regeneratePrometheusRules(r)

	writeJSON(w, http.StatusCreated, rule)
}

func (h *RuleHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" {
		ruleStatus := models.RuleStatus(status)
		if !ruleStatus.IsValid() {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid rule status",
			})
			return
		}

		rules, err := h.repo.ListByStatus(r.Context(), ruleStatus)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, rules)
		return
	}

	rules, err := h.repo.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, rules)
}

func (h *RuleHandler) ListEnabledRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.repo.ListEnabled(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, rules)
}

func (h *RuleHandler) GetRuleByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	rule, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "rule not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

func (h *RuleHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	oldRule, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "rule not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if oldRule.Status == models.RuleStatusArchived {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "archived rules cannot be updated",
		})
		return
	}

	var rule models.Rule

	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if rule.Status == "" {
		rule.Status = oldRule.Status
	}

	if !rule.Status.IsValid() {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid rule status",
		})
		return
	}

	if !isValidRule(rule) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "name, metric, operator and severity are required",
		})
		return
	}

	if !isSupportedMetric(rule.Metric) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "unsupported metric",
		})
		return
	}

	if !isSupportedOperator(rule.Operator) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "unsupported operator",
		})
		return
	}

	rule.Enabled = rule.Status == models.RuleStatusActive

	if err := h.repo.Update(r.Context(), id, &rule); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "rule not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	rule.ID = oldRule.ID
	rule.CreatedAt = oldRule.CreatedAt

	ruleID := oldRule.ID

	if err := h.repo.CreateAuditLog(
		r.Context(),
		&ruleID,
		"RULE_UPDATED",
		rule.Name,
		oldRule,
		rule,
		changedBy(r),
	); err != nil {
		log.Printf("failed to create rule audit log: %v", err)
	}

	h.regeneratePrometheusRules(r)

	writeJSON(w, http.StatusOK, rule)
}

func (h *RuleHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	oldRule, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "rule not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	ruleID := oldRule.ID

	if err := h.repo.CreateAuditLog(
		r.Context(),
		&ruleID,
		"RULE_DELETED",
		oldRule.Name,
		oldRule,
		nil,
		changedBy(r),
	); err != nil {
		log.Printf("failed to create rule audit log: %v", err)
	}

	h.regeneratePrometheusRules(r)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "rule deleted successfully",
	})
}

func (h *RuleHandler) ActivateRule(w http.ResponseWriter, r *http.Request) {
	h.changeRuleStatus(w, r, models.RuleStatusActive, "RULE_ACTIVATED")
}

func (h *RuleHandler) DisableRule(w http.ResponseWriter, r *http.Request) {
	h.changeRuleStatus(w, r, models.RuleStatusDisabled, "RULE_DISABLED")
}

func (h *RuleHandler) ArchiveRule(w http.ResponseWriter, r *http.Request) {
	h.changeRuleStatus(w, r, models.RuleStatusArchived, "RULE_ARCHIVED")
}

func (h *RuleHandler) changeRuleStatus(w http.ResponseWriter, r *http.Request, status models.RuleStatus, action string) {
	id := chi.URLParam(r, "id")

	oldRule, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "rule not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if oldRule.Status == models.RuleStatusArchived {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "archived rules cannot be changed",
		})
		return
	}

	if status == models.RuleStatusActive {
		if !isValidRule(*oldRule) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "rule is invalid and cannot be activated",
			})
			return
		}

		if !isSupportedMetric(oldRule.Metric) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "unsupported metric",
			})
			return
		}

		if !isSupportedOperator(oldRule.Operator) {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "unsupported operator",
			})
			return
		}
	}

	if err := h.repo.UpdateStatus(r.Context(), id, status); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	newRule, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	ruleID := oldRule.ID

	if err := h.repo.CreateAuditLog(
		r.Context(),
		&ruleID,
		action,
		oldRule.Name,
		oldRule,
		newRule,
		changedBy(r),
	); err != nil {
		log.Printf("failed to create rule audit log: %v", err)
	}

	h.regeneratePrometheusRules(r)

	writeJSON(w, http.StatusOK, newRule)
}

func (h *RuleHandler) ListRuleAuditLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.repo.ListAuditLogs(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, logs)
}

func (h *RuleHandler) ListRuleAuditLogsByRuleID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	logs, err := h.repo.ListAuditLogsByRuleID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, logs)
}

func (h *RuleHandler) regeneratePrometheusRules(r *http.Request) {
	rules, err := h.repo.ListEnabled(r.Context())
	if err != nil {
		log.Printf("failed to list active rules for prometheus generation: %v", err)
		return
	}

	rulesFile := os.Getenv("PROMETHEUS_RULES_FILE")
	if rulesFile == "" {
		rulesFile = "/etc/prometheus/rules/dynamic-rules.yml"
	}

	if err := promrules.WriteRulesFile(rulesFile, rules); err != nil {
		log.Printf("failed to write prometheus rules file: %v", err)
		return
	}

	if err := promrules.Reload(); err != nil {
		log.Printf("failed to reload prometheus: %v", err)
		return
	}

	log.Printf("prometheus dynamic rules regenerated and reloaded using active rules")
}

func changedBy(r *http.Request) string {
	email := r.Header.Get("X-User-Email")
	if email == "" {
		return "system"
	}

	return email
}

func isValidRule(rule models.Rule) bool {
	if rule.Name == "" || rule.Metric == "" || rule.Operator == "" || rule.Severity == "" {
		return false
	}

	if rule.Metric == "status" && rule.Value == "" {
		return false
	}

	return true
}

func isSupportedMetric(metric string) bool {
	switch metric {
	case "temperature", "cpu", "memory", "status":
		return true
	default:
		return false
	}
}

func isSupportedOperator(operator string) bool {
	switch operator {
	case ">", ">=", "<", "<=", "==", "!=":
		return true
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
