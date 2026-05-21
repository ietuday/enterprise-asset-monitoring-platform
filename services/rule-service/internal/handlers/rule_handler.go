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

	if rule.Name == "" || rule.Metric == "" || rule.Operator == "" || rule.Severity == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "name, metric, operator and severity are required",
		})
		return
	}

	if rule.Operator != ">" && rule.Operator != ">=" && rule.Operator != "<" && rule.Operator != "<=" && rule.Operator != "==" && rule.Operator != "!=" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "unsupported operator",
		})
		return
	}

	if err := h.repo.Create(r.Context(), &rule); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	h.regeneratePrometheusRules(r)

	writeJSON(w, http.StatusCreated, rule)
}

func (h *RuleHandler) ListRules(w http.ResponseWriter, r *http.Request) {
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

	var rule models.Rule

	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if rule.Name == "" || rule.Metric == "" || rule.Operator == "" || rule.Severity == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "name, metric, operator and severity are required",
		})
		return
	}

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

	rule.ID = 0

	h.regeneratePrometheusRules(r)
	writeJSON(w, http.StatusOK, rule)
}

func (h *RuleHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	h.regeneratePrometheusRules(r)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "rule deleted successfully",
	})
}

func (h *RuleHandler) regeneratePrometheusRules(r *http.Request) {
	rules, err := h.repo.ListEnabled(r.Context())
	if err != nil {
		log.Printf("failed to list enabled rules for prometheus generation: %v", err)
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

	log.Printf("prometheus dynamic rules regenerated and reloaded")
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
