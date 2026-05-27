package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/smtp"
	"strconv"
	"time"

	"notification-service/internal/models"
)

type Sender interface {
	Send(ctx context.Context, channel models.NotificationChannel, req models.SendNotificationRequest) error
}

type Registry struct {
	email   Sender
	webhook Sender
}

func NewRegistry(email Sender, webhook Sender) *Registry {
	return &Registry{email: email, webhook: webhook}
}

func (r *Registry) Send(ctx context.Context, channel models.NotificationChannel, req models.SendNotificationRequest) error {
	switch channel.Type {
	case models.ChannelTypeEmail:
		return r.email.Send(ctx, channel, req)
	case models.ChannelTypeWebhook:
		return r.webhook.Send(ctx, channel, req)
	default:
		return fmt.Errorf("unsupported channel type %s", channel.Type)
	}
}

type WebhookSender struct {
	client *http.Client
}

func NewWebhookSender(timeout time.Duration) *WebhookSender {
	return &WebhookSender{
		client: &http.Client{Timeout: timeout},
	}
}

func (s *WebhookSender) Send(ctx context.Context, channel models.NotificationChannel, req models.SendNotificationRequest) error {
	body := map[string]any{
		"event_type":  req.EventType,
		"subject":     req.Subject,
		"message":     req.Message,
		"severity":    req.Severity,
		"asset_id":    req.AssetID,
		"alert_id":    req.AlertID,
		"incident_id": req.IncidentID,
		"payload":     json.RawMessage(req.Payload),
		"created_at":  time.Now().UTC(),
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, channel.Target, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

type EmailConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

type EmailSender struct {
	config EmailConfig
}

func NewEmailSender(config EmailConfig) *EmailSender {
	return &EmailSender{config: config}
}

func (s *EmailSender) Send(ctx context.Context, channel models.NotificationChannel, req models.SendNotificationRequest) error {
	if s.config.Host == "" || s.config.From == "" {
		return errors.New("SMTP is not configured")
	}

	port := s.config.Port
	if port == "" {
		port = "587"
	}

	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("invalid SMTP port: %s", port)
	}

	message := bytes.Buffer{}
	message.WriteString("From: " + s.config.From + "\r\n")
	message.WriteString("To: " + channel.Target + "\r\n")
	message.WriteString("Subject: " + req.Subject + "\r\n")
	message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	message.WriteString("\r\n")
	message.WriteString(req.Message)
	message.WriteString("\r\n")

	addr := s.config.Host + ":" + port
	var auth smtp.Auth
	if s.config.User != "" || s.config.Password != "" {
		auth = smtp.PlainAuth("", s.config.User, s.config.Password, s.config.Host)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(addr, auth, s.config.From, []string{channel.Target}, message.Bytes())
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}
