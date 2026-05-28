package sla

import (
	"context"
	"log"
	"strconv"
	"time"

	"alert-service/internal/models"
	"alert-service/internal/notification"
)

type Repository interface {
	ListSLARecordsDueForCheck(ctx context.Context) ([]models.IncidentSLATracking, error)
	GetIncidentByID(ctx context.Context, id string) (*models.Incident, error)
	GetEnabledSLAPolicyBySeverity(ctx context.Context, severity string) (*models.SLAPolicy, error)
	UpdateSLAStatus(ctx context.Context, incidentID int64, status string) error
	CreateEscalationHistory(ctx context.Context, escalation *models.EscalationHistory) error
	ExistsEscalationForIncidentAction(ctx context.Context, incidentID int64, action string) (bool, error)
}

type NotificationClient interface {
	Send(ctx context.Context, req notification.SendRequest)
}

type Worker struct {
	repo         Repository
	notification NotificationClient
	interval     time.Duration
}

func NewWorker(repo Repository, notificationClient NotificationClient, interval time.Duration) *Worker {
	return &Worker{
		repo:         repo,
		notification: notificationClient,
		interval:     interval,
	}
}

func (w *Worker) Start(ctx context.Context) {
	if w.interval <= 0 {
		w.interval = time.Minute
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Check(ctx); err != nil {
				log.Printf("SLA worker check failed: %v", err)
			}
		}
	}
}

func (w *Worker) Check(ctx context.Context) error {
	records, err := w.repo.ListSLARecordsDueForCheck(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, record := range records {
		if record.AcknowledgedAt == nil && record.AcknowledgeDueAt != nil && now.After(*record.AcknowledgeDueAt) {
			if err := w.escalate(ctx, record, models.EscalationActionSLAAckBreached, models.SLAStatusAckBreached, "Acknowledgement SLA breached"); err != nil {
				log.Printf("failed to process acknowledgement SLA breach for incident %d: %v", record.IncidentID, err)
			}
		}

		if record.ResolvedAt == nil && record.ResolveDueAt != nil && now.After(*record.ResolveDueAt) {
			if err := w.escalate(ctx, record, models.EscalationActionSLAResolutionBreached, models.SLAStatusResolutionBreached, "Resolution SLA breached"); err != nil {
				log.Printf("failed to process resolution SLA breach for incident %d: %v", record.IncidentID, err)
			}
		}
	}

	return nil
}

func (w *Worker) escalate(ctx context.Context, record models.IncidentSLATracking, action string, status string, reason string) error {
	exists, err := w.repo.ExistsEscalationForIncidentAction(ctx, record.IncidentID, action)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	policy, err := w.repo.GetEnabledSLAPolicyBySeverity(ctx, record.Severity)
	if err != nil {
		return err
	}

	escalation := models.EscalationHistory{
		IncidentID: record.IncidentID,
		Action:     action,
		Reason:     reason,
		Target:     policy.EscalationTarget,
		Actor:      "sla-worker",
	}
	if err := w.repo.CreateEscalationHistory(ctx, &escalation); err != nil {
		return err
	}
	if err := w.repo.UpdateSLAStatus(ctx, record.IncidentID, status); err != nil {
		return err
	}

	incident, err := w.repo.GetIncidentByID(ctx, strconv.FormatInt(record.IncidentID, 10))
	if err != nil {
		return err
	}
	w.notify(ctx, *incident, escalation)

	return nil
}

func (w *Worker) notify(ctx context.Context, incident models.Incident, escalation models.EscalationHistory) {
	if w.notification == nil {
		return
	}

	subject := "Acknowledgement SLA breached"
	message := "Incident #" + strconv.FormatInt(incident.ID, 10) + " was not acknowledged within SLA."
	if escalation.Action == models.EscalationActionSLAResolutionBreached {
		subject = "Resolution SLA breached"
		message = "Incident #" + strconv.FormatInt(incident.ID, 10) + " was not resolved within SLA."
	}

	w.notification.Send(ctx, notification.SendRequest{
		EventType:  escalation.Action,
		Subject:    subject,
		Message:    message,
		Severity:   incident.Severity,
		AssetID:    incident.AssetID,
		AlertID:    incident.AlertID,
		IncidentID: notification.IncidentID(incident.ID),
		Payload: map[string]any{
			"target": escalation.Target,
			"reason": escalation.Reason,
		},
	})
}
