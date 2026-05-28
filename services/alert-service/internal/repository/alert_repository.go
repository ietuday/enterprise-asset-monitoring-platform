package repository

import (
	"context"
	"errors"
	"strings"

	"alert-service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertRepository struct {
	db *pgxpool.Pool
}

var ErrConflict = errors.New("conflict")

func NewAlertRepository(db *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{db: db}
}

func (r *AlertRepository) Create(ctx context.Context, alert *models.Alert) error {
	existingAlert, err := r.FindActiveByAssetAndName(ctx, alert.AssetID, alert.Name)
	if err == nil && existingAlert != nil {
		*alert = *existingAlert
		return nil
	}

	query := `
		INSERT INTO alerts (asset_id, name, severity, status, message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at;
	`

	return r.db.QueryRow(
		ctx,
		query,
		alert.AssetID,
		alert.Name,
		alert.Severity,
		alert.Status,
		alert.Message,
	).Scan(&alert.ID, &alert.CreatedAt, &alert.UpdatedAt)
}

func (r *AlertRepository) FindActiveByAssetAndName(ctx context.Context, assetID string, name string) (*models.Alert, error) {
	var alert models.Alert

	query := `
		SELECT id, asset_id, name, severity, status, message, created_at, updated_at, resolved_at
		FROM alerts
		WHERE asset_id = $1
		AND name = $2
		AND status IN ('OPEN', 'ACKNOWLEDGED')
		ORDER BY created_at DESC
		LIMIT 1;
	`

	err := r.db.QueryRow(ctx, query, assetID, name).Scan(
		&alert.ID,
		&alert.AssetID,
		&alert.Name,
		&alert.Severity,
		&alert.Status,
		&alert.Message,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&alert.ResolvedAt,
	)

	if err != nil {
		return nil, err
	}

	return &alert, nil
}

func (r *AlertRepository) List(ctx context.Context) ([]models.Alert, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, asset_id, name, severity, status, message, created_at, updated_at, resolved_at
		FROM alerts
		ORDER BY created_at DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]models.Alert, 0)

	for rows.Next() {
		var alert models.Alert

		if err := rows.Scan(
			&alert.ID,
			&alert.AssetID,
			&alert.Name,
			&alert.Severity,
			&alert.Status,
			&alert.Message,
			&alert.CreatedAt,
			&alert.UpdatedAt,
			&alert.ResolvedAt,
		); err != nil {
			return nil, err
		}

		alerts = append(alerts, alert)
	}

	return alerts, rows.Err()
}

func (r *AlertRepository) GetByID(ctx context.Context, id string) (*models.Alert, error) {
	var alert models.Alert

	query := `
		SELECT id, asset_id, name, severity, status, message, created_at, updated_at, resolved_at
		FROM alerts
		WHERE id = $1;
	`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&alert.ID,
		&alert.AssetID,
		&alert.Name,
		&alert.Severity,
		&alert.Status,
		&alert.Message,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&alert.ResolvedAt,
	)

	if err != nil {
		return nil, err
	}

	return &alert, nil
}

func (r *AlertRepository) Acknowledge(ctx context.Context, id string) (*models.Alert, error) {
	var alert models.Alert

	query := `
		UPDATE alerts
		SET status = 'ACKNOWLEDGED', updated_at = NOW()
		WHERE id = $1
		RETURNING id, asset_id, name, severity, status, message, created_at, updated_at, resolved_at;
	`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&alert.ID,
		&alert.AssetID,
		&alert.Name,
		&alert.Severity,
		&alert.Status,
		&alert.Message,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&alert.ResolvedAt,
	)

	if err != nil {
		return nil, err
	}

	return &alert, nil
}

func (r *AlertRepository) Resolve(ctx context.Context, id string) (*models.Alert, error) {
	var alert models.Alert

	query := `
		UPDATE alerts
		SET status = 'RESOLVED', updated_at = NOW(), resolved_at = NOW()
		WHERE id = $1
		RETURNING id, asset_id, name, severity, status, message, created_at, updated_at, resolved_at;
	`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&alert.ID,
		&alert.AssetID,
		&alert.Name,
		&alert.Severity,
		&alert.Status,
		&alert.Message,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&alert.ResolvedAt,
	)

	if err != nil {
		return nil, err
	}

	return &alert, nil
}

func (r *AlertRepository) ResolveActiveByAssetAndName(ctx context.Context, assetID string, name string) (*models.Alert, error) {
	var alert models.Alert

	query := `
		UPDATE alerts
		SET status = 'RESOLVED', updated_at = NOW(), resolved_at = NOW()
		WHERE id = (
			SELECT id
			FROM alerts
			WHERE asset_id = $1
			AND name = $2
			AND status IN ('OPEN', 'ACKNOWLEDGED')
			ORDER BY created_at DESC
			LIMIT 1
		)
		RETURNING id, asset_id, name, severity, status, message, created_at, updated_at, resolved_at;
	`

	err := r.db.QueryRow(ctx, query, assetID, name).Scan(
		&alert.ID,
		&alert.AssetID,
		&alert.Name,
		&alert.Severity,
		&alert.Status,
		&alert.Message,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&alert.ResolvedAt,
	)

	if err != nil {
		return nil, err
	}

	return &alert, nil
}

func (r *AlertRepository) CreateIncident(ctx context.Context, incident *models.Incident, actor string, comment string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if incident.Status == "" {
		incident.Status = models.IncidentStatusOpen
	}

	if incident.AlertID != nil {
		existing, err := r.findActiveIncidentByAlertID(ctx, tx, *incident.AlertID)
		if err == nil {
			*incident = *existing
			return tx.Commit(ctx)
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO incidents (alert_id, asset_id, title, description, severity, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (alert_id)
		WHERE alert_id IS NOT NULL AND status IN ('OPEN', 'ASSIGNED', 'ACKNOWLEDGED')
		DO NOTHING
		RETURNING id, alert_id, asset_id, title, description, severity, status, assigned_to,
			resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at;
	`, incident.AlertID, incident.AssetID, incident.Title, incident.Description, incident.Severity, incident.Status).Scan(
		&incident.ID,
		&incident.AlertID,
		&incident.AssetID,
		&incident.Title,
		&incident.Description,
		&incident.Severity,
		&incident.Status,
		&incident.AssignedTo,
		&incident.ResolutionNote,
		&incident.CreatedAt,
		&incident.UpdatedAt,
		&incident.AcknowledgedAt,
		&incident.ResolvedAt,
		&incident.ClosedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) && incident.AlertID != nil {
		existing, findErr := r.findActiveIncidentByAlertID(ctx, tx, *incident.AlertID)
		if findErr != nil {
			return findErr
		}

		*incident = *existing
		return tx.Commit(ctx)
	}
	if err != nil {
		return err
	}

	if actor == "" {
		actor = "system"
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO incident_history (incident_id, action, old_status, new_status, actor, comment)
		VALUES ($1, 'CREATED', NULL, $2, $3, $4);
	`, incident.ID, incident.Status, actor, comment)
	if err != nil {
		return err
	}

	if err := r.createSLATrackingForIncident(ctx, tx, incident); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *AlertRepository) findActiveIncidentByAlertID(ctx context.Context, tx pgx.Tx, alertID int64) (*models.Incident, error) {
	return scanIncident(tx.QueryRow(ctx, `
		SELECT id, alert_id, asset_id, title, description, severity, status, assigned_to,
			resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at
		FROM incidents
		WHERE alert_id = $1
		AND status IN ('OPEN', 'ASSIGNED', 'ACKNOWLEDGED')
		ORDER BY created_at DESC
		LIMIT 1;
	`, alertID))
}

func (r *AlertRepository) ListIncidents(ctx context.Context, filters models.IncidentFilters) ([]models.Incident, error) {
	query := `
		SELECT id, alert_id, asset_id, title, description, severity, status, assigned_to,
			resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at
		FROM incidents
		WHERE ($1 = '' OR status = $1)
		AND ($2 = '' OR severity = $2)
		AND ($3 = '' OR assigned_to = $3)
		ORDER BY created_at DESC;
	`

	rows, err := r.db.Query(ctx, query, filters.Status, filters.Severity, filters.AssignedTo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	incidents := make([]models.Incident, 0)
	for rows.Next() {
		incident, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}

		incidents = append(incidents, *incident)
	}

	return incidents, rows.Err()
}

func (r *AlertRepository) GetIncidentByID(ctx context.Context, id string) (*models.Incident, error) {
	return scanIncident(r.db.QueryRow(ctx, `
		SELECT id, alert_id, asset_id, title, description, severity, status, assigned_to,
			resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at
		FROM incidents
		WHERE id = $1;
	`, id))
}

func (r *AlertRepository) AssignIncident(ctx context.Context, id string, assignedTo string, actor string, comment string) (*models.Incident, error) {
	return r.transitionIncident(ctx, id, models.IncidentStatusAssigned, actor, comment, func(ctx context.Context, tx pgx.Tx, oldStatus string) (*models.Incident, error) {
		return scanIncident(tx.QueryRow(ctx, `
			UPDATE incidents
			SET status = 'ASSIGNED', assigned_to = $2, updated_at = NOW()
			WHERE id = $1
			RETURNING id, alert_id, asset_id, title, description, severity, status, assigned_to,
				resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at;
		`, id, assignedTo))
	})
}

func (r *AlertRepository) AcknowledgeIncident(ctx context.Context, id string, actor string, comment string) (*models.Incident, error) {
	incident, err := r.transitionIncident(ctx, id, models.IncidentStatusAcknowledged, actor, comment, func(ctx context.Context, tx pgx.Tx, oldStatus string) (*models.Incident, error) {
		return scanIncident(tx.QueryRow(ctx, `
			UPDATE incidents
			SET status = 'ACKNOWLEDGED', acknowledged_at = COALESCE(acknowledged_at, NOW()), updated_at = NOW()
			WHERE id = $1
			RETURNING id, alert_id, asset_id, title, description, severity, status, assigned_to,
				resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at;
		`, id))
	})
	if err != nil {
		return nil, err
	}

	if err := r.UpdateAcknowledgedAt(ctx, incident.ID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	return incident, nil
}

func (r *AlertRepository) ResolveIncident(ctx context.Context, id string, actor string, resolutionNote string) (*models.Incident, error) {
	incident, err := r.transitionIncident(ctx, id, models.IncidentStatusResolved, actor, resolutionNote, func(ctx context.Context, tx pgx.Tx, oldStatus string) (*models.Incident, error) {
		return scanIncident(tx.QueryRow(ctx, `
			UPDATE incidents
			SET status = 'RESOLVED', resolution_note = $2, resolved_at = NOW(), updated_at = NOW()
			WHERE id = $1
			RETURNING id, alert_id, asset_id, title, description, severity, status, assigned_to,
				resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at;
		`, id, resolutionNote))
	})
	if err != nil {
		return nil, err
	}

	if err := r.UpdateResolvedAt(ctx, incident.ID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	return incident, nil
}

func (r *AlertRepository) CloseIncident(ctx context.Context, id string, actor string, comment string) (*models.Incident, error) {
	return r.transitionIncident(ctx, id, models.IncidentStatusClosed, actor, comment, func(ctx context.Context, tx pgx.Tx, oldStatus string) (*models.Incident, error) {
		return scanIncident(tx.QueryRow(ctx, `
			UPDATE incidents
			SET status = 'CLOSED', closed_at = NOW(), updated_at = NOW()
			WHERE id = $1
			RETURNING id, alert_id, asset_id, title, description, severity, status, assigned_to,
				resolution_note, created_at, updated_at, acknowledged_at, resolved_at, closed_at;
		`, id))
	})
}

func (r *AlertRepository) GetIncidentHistory(ctx context.Context, incidentID string) ([]models.IncidentHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, incident_id, action, old_status, new_status, actor, comment, created_at
		FROM incident_history
		WHERE incident_id = $1
		ORDER BY created_at ASC, id ASC;
	`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]models.IncidentHistory, 0)
	for rows.Next() {
		item, err := scanIncidentHistory(rows)
		if err != nil {
			return nil, err
		}

		history = append(history, *item)
	}

	return history, rows.Err()
}

func (r *AlertRepository) AddIncidentHistory(ctx context.Context, history *models.IncidentHistory) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO incident_history (incident_id, action, old_status, new_status, actor, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at;
	`, history.IncidentID, history.Action, history.OldStatus, history.NewStatus, history.Actor, history.Comment).Scan(&history.ID, &history.CreatedAt)
}

func (r *AlertRepository) CreateSLAPolicy(ctx context.Context, policy *models.SLAPolicy) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO sla_policies (severity, acknowledge_within_minutes, resolve_within_minutes, escalation_target, enabled)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at;
	`, policy.Severity, policy.AcknowledgeWithinMinutes, policy.ResolveWithinMinutes, policy.EscalationTarget, policy.Enabled).Scan(
		&policy.ID, &policy.CreatedAt, &policy.UpdatedAt,
	)
	if isUniqueViolation(err) {
		return ErrConflict
	}
	return err
}

func (r *AlertRepository) ListSLAPolicies(ctx context.Context) ([]models.SLAPolicy, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, severity, acknowledge_within_minutes, resolve_within_minutes, escalation_target, enabled, created_at, updated_at
		FROM sla_policies
		ORDER BY CASE severity WHEN 'CRITICAL' THEN 1 WHEN 'HIGH' THEN 2 WHEN 'MEDIUM' THEN 3 WHEN 'LOW' THEN 4 ELSE 5 END, severity;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	policies := make([]models.SLAPolicy, 0)
	for rows.Next() {
		policy, err := scanSLAPolicy(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, *policy)
	}
	return policies, rows.Err()
}

func (r *AlertRepository) GetSLAPolicyByID(ctx context.Context, id string) (*models.SLAPolicy, error) {
	return scanSLAPolicy(r.db.QueryRow(ctx, `
		SELECT id, severity, acknowledge_within_minutes, resolve_within_minutes, escalation_target, enabled, created_at, updated_at
		FROM sla_policies
		WHERE id = $1;
	`, id))
}

func (r *AlertRepository) GetEnabledSLAPolicyBySeverity(ctx context.Context, severity string) (*models.SLAPolicy, error) {
	return scanSLAPolicy(r.db.QueryRow(ctx, `
		SELECT id, severity, acknowledge_within_minutes, resolve_within_minutes, escalation_target, enabled, created_at, updated_at
		FROM sla_policies
		WHERE severity = $1 AND enabled = TRUE;
	`, strings.ToUpper(severity)))
}

func (r *AlertRepository) UpdateSLAPolicy(ctx context.Context, policy *models.SLAPolicy) error {
	err := r.db.QueryRow(ctx, `
		UPDATE sla_policies
		SET severity = $2, acknowledge_within_minutes = $3, resolve_within_minutes = $4,
			escalation_target = $5, enabled = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING created_at, updated_at;
	`, policy.ID, policy.Severity, policy.AcknowledgeWithinMinutes, policy.ResolveWithinMinutes, policy.EscalationTarget, policy.Enabled).Scan(
		&policy.CreatedAt, &policy.UpdatedAt,
	)
	if isUniqueViolation(err) {
		return ErrConflict
	}
	return err
}

func (r *AlertRepository) DeleteSLAPolicy(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM sla_policies WHERE id = $1;`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *AlertRepository) CreateIncidentSLATracking(ctx context.Context, incident *models.Incident) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := r.createSLATrackingForIncident(ctx, tx, incident); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *AlertRepository) GetIncidentSLA(ctx context.Context, incidentID string) (*models.IncidentSLATracking, error) {
	tracking, err := r.GetIncidentSLAByIncidentID(ctx, incidentID)
	if err == nil {
		return tracking, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	incident, incidentErr := r.GetIncidentByID(ctx, incidentID)
	if incidentErr != nil {
		return nil, incidentErr
	}
	if err := r.CreateIncidentSLATracking(ctx, incident); err != nil {
		return nil, err
	}

	return r.GetIncidentSLAByIncidentID(ctx, incidentID)
}

func (r *AlertRepository) GetIncidentSLAByIncidentID(ctx context.Context, incidentID string) (*models.IncidentSLATracking, error) {
	return scanSLATracking(r.db.QueryRow(ctx, `
		SELECT id, incident_id, severity, status, acknowledge_due_at, resolve_due_at,
			acknowledged_at, resolved_at, escalated_at, created_at, updated_at
		FROM incident_sla_tracking
		WHERE incident_id = $1;
	`, incidentID))
}

func (r *AlertRepository) UpdateAcknowledgedAt(ctx context.Context, incidentID int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE incident_sla_tracking
		SET acknowledged_at = COALESCE(acknowledged_at, NOW()),
			status = CASE
				WHEN status IN ('NO_POLICY', 'RESOLUTION_BREACHED', 'ESCALATED', 'COMPLETED') THEN status
				WHEN acknowledge_due_at IS NOT NULL AND NOW() > acknowledge_due_at THEN 'ACK_BREACHED'
				ELSE status
			END,
			updated_at = NOW()
		WHERE incident_id = $1;
	`, incidentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *AlertRepository) UpdateResolvedAt(ctx context.Context, incidentID int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE incident_sla_tracking
		SET resolved_at = COALESCE(resolved_at, NOW()),
			status = CASE
				WHEN status = 'NO_POLICY' THEN status
				WHEN resolve_due_at IS NOT NULL AND NOW() > resolve_due_at THEN 'RESOLUTION_BREACHED'
				WHEN status = 'ACK_BREACHED' THEN status
				WHEN status = 'ESCALATED' THEN status
				ELSE 'COMPLETED'
			END,
			updated_at = NOW()
		WHERE incident_id = $1;
	`, incidentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *AlertRepository) UpdateSLAStatus(ctx context.Context, incidentID int64, status string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE incident_sla_tracking
		SET status = $2, updated_at = NOW()
		WHERE incident_id = $1;
	`, incidentID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *AlertRepository) MarkEscalated(ctx context.Context, incidentID int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE incident_sla_tracking
		SET status = 'ESCALATED', escalated_at = COALESCE(escalated_at, NOW()), updated_at = NOW()
		WHERE incident_id = $1;
	`, incidentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *AlertRepository) ListSLABreaches(ctx context.Context, filters models.SLABreachFilters) ([]models.IncidentSLATracking, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, incident_id, severity, status, acknowledge_due_at, resolve_due_at,
			acknowledged_at, resolved_at, escalated_at, created_at, updated_at
		FROM incident_sla_tracking
		WHERE status IN ('ACK_BREACHED', 'RESOLUTION_BREACHED', 'ESCALATED')
		AND ($1 = '' OR status = $1)
		AND ($2 = '' OR severity = $2)
		AND ($3 = '' OR incident_id::TEXT = $3)
		ORDER BY updated_at DESC, id DESC;
	`, filters.Status, filters.Severity, filters.IncidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.IncidentSLATracking, 0)
	for rows.Next() {
		item, err := scanSLATracking(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *AlertRepository) ListSLARecordsDueForCheck(ctx context.Context) ([]models.IncidentSLATracking, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.incident_id, s.severity, s.status, s.acknowledge_due_at, s.resolve_due_at,
			s.acknowledged_at, s.resolved_at, s.escalated_at, s.created_at, s.updated_at
		FROM incident_sla_tracking s
		JOIN incidents i ON i.id = s.incident_id
		WHERE i.status NOT IN ('RESOLVED', 'CLOSED')
		AND s.status NOT IN ('COMPLETED', 'NO_POLICY')
		AND (
			(s.acknowledged_at IS NULL AND s.acknowledge_due_at IS NOT NULL AND NOW() > s.acknowledge_due_at)
			OR (s.resolved_at IS NULL AND s.resolve_due_at IS NOT NULL AND NOW() > s.resolve_due_at)
		)
		ORDER BY s.updated_at ASC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.IncidentSLATracking, 0)
	for rows.Next() {
		item, err := scanSLATracking(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *AlertRepository) CreateEscalationHistory(ctx context.Context, escalation *models.EscalationHistory) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO escalation_history (incident_id, action, reason, target, actor)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at;
	`, escalation.IncidentID, escalation.Action, escalation.Reason, escalation.Target, escalation.Actor).Scan(&escalation.ID, &escalation.CreatedAt)
}

func (r *AlertRepository) ListEscalationsByIncidentID(ctx context.Context, incidentID string) ([]models.EscalationHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, incident_id, action, reason, target, actor, created_at
		FROM escalation_history
		WHERE incident_id = $1
		ORDER BY created_at ASC, id ASC;
	`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.EscalationHistory, 0)
	for rows.Next() {
		item, err := scanEscalationHistory(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *AlertRepository) ExistsEscalationForIncidentAction(ctx context.Context, incidentID int64, action string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM escalation_history WHERE incident_id = $1 AND action = $2
		);
	`, incidentID, action).Scan(&exists)
	return exists, err
}

func (r *AlertRepository) transitionIncident(
	ctx context.Context,
	id string,
	newStatus string,
	actor string,
	comment string,
	update func(context.Context, pgx.Tx, string) (*models.Incident, error),
) (*models.Incident, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var oldStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM incidents WHERE id = $1;`, id).Scan(&oldStatus); err != nil {
		return nil, err
	}

	incident, err := update(ctx, tx, oldStatus)
	if err != nil {
		return nil, err
	}

	if actor == "" {
		actor = "system"
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO incident_history (incident_id, action, old_status, new_status, actor, comment)
		VALUES ($1, $2, $3, $4, $5, $6);
	`, incident.ID, newStatus, oldStatus, newStatus, actor, comment)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return incident, nil
}

func (r *AlertRepository) createSLATrackingForIncident(ctx context.Context, tx pgx.Tx, incident *models.Incident) error {
	var policy models.SLAPolicy
	err := tx.QueryRow(ctx, `
		SELECT id, severity, acknowledge_within_minutes, resolve_within_minutes, escalation_target, enabled, created_at, updated_at
		FROM sla_policies
		WHERE severity = $1 AND enabled = TRUE;
	`, incident.Severity).Scan(
		&policy.ID,
		&policy.Severity,
		&policy.AcknowledgeWithinMinutes,
		&policy.ResolveWithinMinutes,
		&policy.EscalationTarget,
		&policy.Enabled,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `
			INSERT INTO incident_sla_tracking (incident_id, severity, status)
			VALUES ($1, $2, 'NO_POLICY')
			ON CONFLICT (incident_id) DO NOTHING;
		`, incident.ID, incident.Severity)
		return err
	}
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO incident_sla_tracking (incident_id, severity, status, acknowledge_due_at, resolve_due_at)
		VALUES ($1, $2, 'ON_TRACK', $3::timestamp + ($4 || ' minutes')::interval, $3::timestamp + ($5 || ' minutes')::interval)
		ON CONFLICT (incident_id) DO NOTHING;
	`, incident.ID, incident.Severity, incident.CreatedAt, policy.AcknowledgeWithinMinutes, policy.ResolveWithinMinutes)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanIncident(row rowScanner) (*models.Incident, error) {
	var incident models.Incident

	err := row.Scan(
		&incident.ID,
		&incident.AlertID,
		&incident.AssetID,
		&incident.Title,
		&incident.Description,
		&incident.Severity,
		&incident.Status,
		&incident.AssignedTo,
		&incident.ResolutionNote,
		&incident.CreatedAt,
		&incident.UpdatedAt,
		&incident.AcknowledgedAt,
		&incident.ResolvedAt,
		&incident.ClosedAt,
	)
	if err != nil {
		return nil, err
	}

	return &incident, nil
}

func scanIncidentHistory(row rowScanner) (*models.IncidentHistory, error) {
	var history models.IncidentHistory

	err := row.Scan(
		&history.ID,
		&history.IncidentID,
		&history.Action,
		&history.OldStatus,
		&history.NewStatus,
		&history.Actor,
		&history.Comment,
		&history.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &history, nil
}

func scanSLAPolicy(row rowScanner) (*models.SLAPolicy, error) {
	var policy models.SLAPolicy
	err := row.Scan(
		&policy.ID,
		&policy.Severity,
		&policy.AcknowledgeWithinMinutes,
		&policy.ResolveWithinMinutes,
		&policy.EscalationTarget,
		&policy.Enabled,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func scanSLATracking(row rowScanner) (*models.IncidentSLATracking, error) {
	var tracking models.IncidentSLATracking
	err := row.Scan(
		&tracking.ID,
		&tracking.IncidentID,
		&tracking.Severity,
		&tracking.Status,
		&tracking.AcknowledgeDueAt,
		&tracking.ResolveDueAt,
		&tracking.AcknowledgedAt,
		&tracking.ResolvedAt,
		&tracking.EscalatedAt,
		&tracking.CreatedAt,
		&tracking.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &tracking, nil
}

func scanEscalationHistory(row rowScanner) (*models.EscalationHistory, error) {
	var escalation models.EscalationHistory
	err := row.Scan(
		&escalation.ID,
		&escalation.IncidentID,
		&escalation.Action,
		&escalation.Reason,
		&escalation.Target,
		&escalation.Actor,
		&escalation.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &escalation, nil
}
