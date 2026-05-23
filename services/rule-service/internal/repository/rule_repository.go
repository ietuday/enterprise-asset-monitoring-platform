package repository

import (
	"context"
	"encoding/json"

	"rule-service/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RuleRepository struct {
	pool *pgxpool.Pool
}

func NewRuleRepository(pool *pgxpool.Pool) *RuleRepository {
	return &RuleRepository{pool: pool}
}

func (r *RuleRepository) Create(ctx context.Context, rule *models.Rule) error {
	if rule.Status == "" {
		rule.Status = models.RuleStatusDraft
	}

	rule.Enabled = rule.Status == models.RuleStatusActive

	query := `
	INSERT INTO monitoring_rules (name, metric, operator, threshold, value, severity, enabled, status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id, created_at, updated_at;
	`

	return r.pool.QueryRow(
		ctx,
		query,
		rule.Name,
		rule.Metric,
		rule.Operator,
		rule.Threshold,
		rule.Value,
		rule.Severity,
		rule.Enabled,
		rule.Status,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

func (r *RuleRepository) List(ctx context.Context) ([]models.Rule, error) {
	query := `
	SELECT id, name, metric, operator, threshold, COALESCE(value, ''), severity, enabled,
	       COALESCE(status, 'draft'), created_at, updated_at
	FROM monitoring_rules
	ORDER BY id DESC;
	`

	return r.scanRules(ctx, query)
}

func (r *RuleRepository) ListByStatus(ctx context.Context, status models.RuleStatus) ([]models.Rule, error) {
	query := `
	SELECT id, name, metric, operator, threshold, COALESCE(value, ''), severity, enabled,
	       COALESCE(status, 'draft'), created_at, updated_at
	FROM monitoring_rules
	WHERE status = $1
	ORDER BY id ASC;
	`

	return r.scanRules(ctx, query, status)
}

func (r *RuleRepository) GetByID(ctx context.Context, id string) (*models.Rule, error) {
	query := `
	SELECT id, name, metric, operator, threshold, COALESCE(value, ''), severity, enabled,
	       COALESCE(status, 'draft'), created_at, updated_at
	FROM monitoring_rules
	WHERE id = $1;
	`

	var rule models.Rule

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Metric,
		&rule.Operator,
		&rule.Threshold,
		&rule.Value,
		&rule.Severity,
		&rule.Enabled,
		&rule.Status,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &rule, nil
}

func (r *RuleRepository) Update(ctx context.Context, id string, rule *models.Rule) error {
	if rule.Status == "" {
		rule.Status = models.RuleStatusDraft
	}

	rule.Enabled = rule.Status == models.RuleStatusActive

	query := `
	UPDATE monitoring_rules
	SET name = $1,
	    metric = $2,
	    operator = $3,
	    threshold = $4,
	    value = $5,
	    severity = $6,
	    enabled = $7,
	    status = $8,
	    updated_at = NOW()
	WHERE id = $9
	RETURNING updated_at;
	`

	return r.pool.QueryRow(
		ctx,
		query,
		rule.Name,
		rule.Metric,
		rule.Operator,
		rule.Threshold,
		rule.Value,
		rule.Severity,
		rule.Enabled,
		rule.Status,
		id,
	).Scan(&rule.UpdatedAt)
}

func (r *RuleRepository) UpdateStatus(ctx context.Context, id string, status models.RuleStatus) error {
	enabled := status == models.RuleStatusActive

	query := `
	UPDATE monitoring_rules
	SET status = $1,
	    enabled = $2,
	    updated_at = NOW()
	WHERE id = $3;
	`

	_, err := r.pool.Exec(ctx, query, status, enabled, id)
	return err
}

func (r *RuleRepository) Delete(ctx context.Context, id string) error {
	query := `
	DELETE FROM monitoring_rules
	WHERE id = $1;
	`

	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *RuleRepository) ListEnabled(ctx context.Context) ([]models.Rule, error) {
	return r.ListByStatus(ctx, models.RuleStatusActive)
}

func (r *RuleRepository) scanRules(ctx context.Context, query string, args ...any) ([]models.Rule, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]models.Rule, 0)

	for rows.Next() {
		var rule models.Rule

		if err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Metric,
			&rule.Operator,
			&rule.Threshold,
			&rule.Value,
			&rule.Severity,
			&rule.Enabled,
			&rule.Status,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, err
		}

		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

func (r *RuleRepository) CreateAuditLog(
	ctx context.Context,
	ruleID *int64,
	action string,
	ruleName string,
	oldValue any,
	newValue any,
	changedBy string,
) error {
	var oldJSON []byte
	var newJSON []byte
	var err error

	if oldValue != nil {
		oldJSON, err = json.Marshal(oldValue)
		if err != nil {
			return err
		}
	}

	if newValue != nil {
		newJSON, err = json.Marshal(newValue)
		if err != nil {
			return err
		}
	}

	query := `
	INSERT INTO rule_audit_logs (rule_id, action, rule_name, old_value, new_value, changed_by)
	VALUES ($1, $2, $3, $4, $5, $6);
	`

	_, err = r.pool.Exec(ctx, query, ruleID, action, ruleName, oldJSON, newJSON, changedBy)
	return err
}

func (r *RuleRepository) ListAuditLogs(ctx context.Context) ([]models.RuleAuditLog, error) {
	query := `
	SELECT id, rule_id, action, rule_name, old_value, new_value, changed_by, created_at
	FROM rule_audit_logs
	ORDER BY created_at DESC
	LIMIT 100;
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]models.RuleAuditLog, 0)

	for rows.Next() {
		var item models.RuleAuditLog

		if err := rows.Scan(
			&item.ID,
			&item.RuleID,
			&item.Action,
			&item.RuleName,
			&item.OldValue,
			&item.NewValue,
			&item.ChangedBy,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}

		logs = append(logs, item)
	}

	return logs, rows.Err()
}

func (r *RuleRepository) ListAuditLogsByRuleID(ctx context.Context, ruleID string) ([]models.RuleAuditLog, error) {
	query := `
	SELECT id, rule_id, action, rule_name, old_value, new_value, changed_by, created_at
	FROM rule_audit_logs
	WHERE rule_id = $1
	ORDER BY created_at DESC;
	`

	rows, err := r.pool.Query(ctx, query, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]models.RuleAuditLog, 0)

	for rows.Next() {
		var item models.RuleAuditLog

		if err := rows.Scan(
			&item.ID,
			&item.RuleID,
			&item.Action,
			&item.RuleName,
			&item.OldValue,
			&item.NewValue,
			&item.ChangedBy,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}

		logs = append(logs, item)
	}

	return logs, rows.Err()
}
