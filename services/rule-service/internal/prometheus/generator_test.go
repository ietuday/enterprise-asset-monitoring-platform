package prometheus

import (
	"strings"
	"testing"

	"rule-service/internal/models"
)

func TestGenerateRulesYAMLIncludesOnlyActiveRules(t *testing.T) {
	rules := []models.Rule{
		{
			Name:      "High CPU",
			Metric:    "cpu",
			Operator:  ">",
			Threshold: 80,
			Severity:  "critical",
			Enabled:   true,
			Status:    models.RuleStatusActive,
		},
		{
			Name:      "High Memory",
			Metric:    "memory",
			Operator:  ">",
			Threshold: 90,
			Severity:  "warning",
			Enabled:   false,
			Status:    models.RuleStatusDraft,
		},
		{
			Name:      "High Temperature",
			Metric:    "temperature",
			Operator:  ">",
			Threshold: 70,
			Severity:  "warning",
			Enabled:   false,
			Status:    models.RuleStatusDisabled,
		},
	}

	content, err := GenerateRulesYAML(rules)
	if err != nil {
		t.Fatalf("GenerateRulesYAML returned error: %v", err)
	}

	yaml := string(content)

	if !strings.Contains(yaml, "HighCPU") {
		t.Fatalf("expected active rule to be included, got: %s", yaml)
	}

	if strings.Contains(yaml, "HighMemory") {
		t.Fatalf("expected draft rule to be excluded, got: %s", yaml)
	}

	if strings.Contains(yaml, "HighTemperature") {
		t.Fatalf("expected disabled rule to be excluded, got: %s", yaml)
	}
}

func TestGenerateRulesYAMLReturnsEmptyRulesWhenNoActiveRules(t *testing.T) {
	rules := []models.Rule{
		{
			Name:      "Draft CPU",
			Metric:    "cpu",
			Operator:  ">",
			Threshold: 80,
			Severity:  "critical",
			Enabled:   false,
			Status:    models.RuleStatusDraft,
		},
		{
			Name:      "Disabled Memory",
			Metric:    "memory",
			Operator:  ">",
			Threshold: 90,
			Severity:  "warning",
			Enabled:   false,
			Status:    models.RuleStatusDisabled,
		},
	}

	content, err := GenerateRulesYAML(rules)
	if err != nil {
		t.Fatalf("GenerateRulesYAML returned error: %v", err)
	}

	yaml := string(content)

	if !strings.Contains(yaml, "      []") {
		t.Fatalf("expected empty rules list when no active rules exist, got: %s", yaml)
	}
}
