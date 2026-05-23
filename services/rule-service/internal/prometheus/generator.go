package prometheus

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"rule-service/internal/models"
)

func metricName(metric string) string {
	switch metric {
	case "temperature":
		return "asset_temperature_celsius"
	case "cpu":
		return "asset_cpu_usage_percent"
	case "memory":
		return "asset_memory_usage_percent"
	case "status":
		return "asset_status"
	default:
		return ""
	}
}

func alertName(name string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	return replacer.Replace(name)
}

func GenerateRulesYAML(rules []models.Rule) ([]byte, error) {
	var buffer bytes.Buffer

	buffer.WriteString("groups:\n")
	buffer.WriteString("  - name: dynamic-monitoring-rules\n")
	buffer.WriteString("    interval: 5s\n")
	buffer.WriteString("    rules:\n")

	activeRuleCount := 0

	for _, rule := range rules {
		if rule.Status != models.RuleStatusActive {
			continue
		}

		promMetric := metricName(rule.Metric)
		if promMetric == "" {
			continue
		}

		activeRuleCount++

		buffer.WriteString(fmt.Sprintf("      - alert: %s\n", alertName(rule.Name)))
		if rule.Metric == "status" {
			buffer.WriteString(fmt.Sprintf("        expr: %s{status=\"%s\"} == 1\n", promMetric, rule.Value))
		} else {
			buffer.WriteString(fmt.Sprintf("        expr: %s %s %.2f\n", promMetric, rule.Operator, rule.Threshold))
		}
		buffer.WriteString("        for: 5s\n")
		buffer.WriteString("        labels:\n")
		buffer.WriteString(fmt.Sprintf("          severity: %s\n", strings.ToLower(rule.Severity)))
		buffer.WriteString("        annotations:\n")
		buffer.WriteString(fmt.Sprintf("          summary: \"%s\"\n", rule.Name))
		buffer.WriteString(fmt.Sprintf("          description: \"Dynamic rule %s triggered for asset {{ $labels.asset_id }}\"\n", rule.Name))
		buffer.WriteString("          asset_id: \"{{ $labels.asset_id }}\"\n")
		buffer.WriteString(fmt.Sprintf("          alert_name: \"%s\"\n", rule.Name))
	}

	if activeRuleCount == 0 {
		buffer.WriteString("      []\n")
	}

	return buffer.Bytes(), nil
}

func WriteRulesFile(path string, rules []models.Rule) error {
	content, err := GenerateRulesYAML(rules)
	if err != nil {
		return err
	}

	return os.WriteFile(path, content, 0644)
}
