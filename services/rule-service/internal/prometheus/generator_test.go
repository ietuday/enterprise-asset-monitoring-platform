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
}
