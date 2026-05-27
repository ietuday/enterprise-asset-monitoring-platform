package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
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

	message, fromAddress, toAddress, err := buildEmailMessage(s.config.From, channel.Target, req.Subject, req.Message)
	if err != nil {
		return err
	}

	addr := s.config.Host + ":" + port
	var auth smtp.Auth
	if s.config.User != "" || s.config.Password != "" {
		auth = smtp.PlainAuth("", s.config.User, s.config.Password, s.config.Host)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(addr, auth, fromAddress, []string{toAddress}, message)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

type emailHeader struct {
	name  string
	value string
}

func buildEmailMessage(from string, to string, subject string, body string) ([]byte, string, string, error) {
	fromAddress, err := validateEmailAddress(from)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid from address: %w", err)
	}

	toAddress, err := validateEmailAddress(to)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid recipient address: %w", err)
	}

	headers := []emailHeader{
		{name: "From", value: sanitizeEmailHeader(from)},
		{name: "To", value: sanitizeEmailHeader(to)},
		{name: "Subject", value: sanitizeEmailHeader(subject)},
		{name: "MIME-Version", value: "1.0"},
		{name: "Content-Type", value: "text/plain; charset=UTF-8"},
	}

	message := bytes.Buffer{}
	for _, header := range headers {
		message.WriteString(header.name)
		message.WriteString(": ")
		message.WriteString(header.value)
		message.WriteString("\r\n")
	}

	message.WriteString("\r\n")
	message.WriteString(sanitizeEmailBody(body))
	message.WriteString("\r\n")

	return message.Bytes(), fromAddress, toAddress, nil
}

func sanitizeEmailHeader(value string) string {
	sanitized := strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
	sanitized = strings.Join(strings.Fields(sanitized), " ")
	if len(sanitized) > 255 {
		return sanitized[:255]
	}

	return sanitized
}

func sanitizeEmailBody(value string) string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	var builder strings.Builder
	builder.Grow(len(normalized))

	count := 0
	for _, char := range normalized {
		if count >= 10000 {
			break
		}

		if char == '\n' || char == '\t' || (char >= 0x20 && char != 0x7f) {
			builder.WriteRune(char)
			count++
		}
	}

	return strings.ReplaceAll(builder.String(), "\n", "\r\n")
}

func validateEmailAddress(value string) (string, error) {
	if strings.ContainsAny(value, "\r\n") {
		return "", errors.New("email address must not contain CR or LF")
	}

	parsed, err := mail.ParseAddress(strings.TrimSpace(value))
	if err != nil {
		return "", err
	}
	if parsed.Address == "" {
		return "", errors.New("email address is required")
	}

	return parsed.Address, nil
}
