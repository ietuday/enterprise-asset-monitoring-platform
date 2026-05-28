package sender

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"notification-service/internal/models"

	gomail "gopkg.in/gomail.v2"
)

func TestEmailSubjectHeaderInjectionIsSanitized(t *testing.T) {
	message, err := newEmailMessage(
		"ops@example.com",
		"recipient@example.com",
		"Alert\r\nBcc: attacker@example.com",
		"Body text",
	)
	if err != nil {
		t.Fatalf("expected message build to succeed: %v", err)
	}

	raw := gomailMessageString(t, message)
	headers := emailHeaders(t, raw)
	if strings.Contains(headers, "\r\nBcc:") || strings.Contains(headers, "\nBcc:") {
		t.Fatalf("expected injected Bcc header to be removed, headers were %q", headers)
	}
	if !strings.Contains(headers, "Subject: Alert Bcc: attacker@example.com") {
		t.Fatalf("expected sanitized subject header, headers were %q", headers)
	}
}

func TestEmailRecipientContainingNewlineIsRejected(t *testing.T) {
	_, err := newEmailMessage(
		"ops@example.com",
		"recipient@example.com\nBcc: attacker@example.com",
		"Alert",
		"Body text",
	)
	if err == nil {
		t.Fatal("expected recipient with newline to be rejected")
	}
}

func TestEmailFromContainingCRLFIsRejected(t *testing.T) {
	_, err := newEmailMessage(
		"ops@example.com\r\nBcc: attacker@example.com",
		"recipient@example.com",
		"Alert",
		"Body text",
	)
	if err == nil {
		t.Fatal("expected from address with CRLF to be rejected")
	}
}

func TestEmailBodyControlCharactersAreRemoved(t *testing.T) {
	body := sanitizeEmailBody("Line\x00 one\tok\x1f\rLine two\x7f")
	if strings.ContainsAny(body, "\x00\x1f\x7f") {
		t.Fatalf("expected control characters to be removed, body was %q", body)
	}
	if !strings.Contains(body, "Line one\tok\nLine two") {
		t.Fatalf("expected tab and normalized newline to remain, body was %q", body)
	}
}

func TestNormalEmailMessageBuildsSafely(t *testing.T) {
	message, err := newEmailMessage(
		"Ops <ops@example.com>",
		"recipient@example.com",
		"Critical incident created",
		"Incident created for asset motor-101",
	)
	if err != nil {
		t.Fatalf("expected message build to succeed: %v", err)
	}

	raw := gomailMessageString(t, message)
	if !strings.Contains(raw, "From: ops@example.com") {
		t.Fatalf("expected sanitized from header, got %q", raw)
	}
	if !strings.Contains(raw, "To: recipient@example.com") {
		t.Fatalf("expected sanitized recipient header, got %q", raw)
	}
	if !strings.Contains(raw, "Subject: Critical incident created") {
		t.Fatalf("expected subject header, got %q", raw)
	}
	if !strings.Contains(raw, "Incident created for asset motor-101") {
		t.Fatalf("expected body text, got %q", raw)
	}
}

func TestEmailBodyCannotInjectHeaders(t *testing.T) {
	message, err := newEmailMessage(
		"ops@example.com",
		"recipient@example.com",
		"Alert",
		"Line one\r\nBcc: attacker@example.com",
	)
	if err != nil {
		t.Fatalf("expected message build to succeed: %v", err)
	}

	raw := gomailMessageString(t, message)
	headers := emailHeaders(t, raw)
	if strings.Contains(headers, "Bcc: attacker@example.com") {
		t.Fatalf("expected body text not to appear in headers, headers were %q", headers)
	}
	if !strings.Contains(raw, "Line one") || !strings.Contains(raw, "Bcc: attacker@example.com") {
		t.Fatalf("expected body text to remain in email body, got %q", raw)
	}
}

func TestEmailSenderMissingSMTPConfigReturnsClearError(t *testing.T) {
	sender := NewEmailSender(EmailConfig{})
	err := sender.Send(context.Background(), models.NotificationChannel{
		Target: "recipient@example.com",
	}, models.SendNotificationRequest{
		Subject: "Alert",
		Message: "Body text",
	})

	if err == nil || err.Error() != "SMTP is not configured" {
		t.Fatalf("expected SMTP is not configured error, got %v", err)
	}
}

func gomailMessageString(t *testing.T, message *gomail.Message) string {
	t.Helper()

	var buffer bytes.Buffer
	if _, err := message.WriteTo(&buffer); err != nil {
		t.Fatalf("failed to render gomail message: %v", err)
	}

	return buffer.String()
}

func emailHeaders(t *testing.T, message string) string {
	t.Helper()

	parts := strings.SplitN(message, "\r\n\r\n", 2)
	if len(parts) != 2 {
		t.Fatalf("expected header/body separator, got %q", message)
	}

	return parts[0]
}
