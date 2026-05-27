package sender

import (
	"strings"
	"testing"
)

func TestEmailSubjectHeaderInjectionIsSanitized(t *testing.T) {
	message, _, _, err := buildEmailMessage(
		"ops@example.com",
		"recipient@example.com",
		"Alert\r\nBcc: attacker@example.com",
		"Body text",
	)
	if err != nil {
		t.Fatalf("expected message build to succeed: %v", err)
	}

	headers := emailHeaders(t, string(message))
	if strings.Contains(headers, "\r\nBcc:") || strings.Contains(headers, "\nBcc:") {
		t.Fatalf("expected injected Bcc header to be removed, headers were %q", headers)
	}
	if !strings.Contains(headers, "Subject: Alert Bcc: attacker@example.com") {
		t.Fatalf("expected sanitized subject header, headers were %q", headers)
	}
}

func TestEmailRecipientContainingNewlineIsRejected(t *testing.T) {
	_, _, _, err := buildEmailMessage(
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
	_, _, _, err := buildEmailMessage(
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
	message, _, _, err := buildEmailMessage(
		"ops@example.com",
		"recipient@example.com",
		"Alert",
		"Line\x00 one\tok\x1f\rLine two\x7f",
	)
	if err != nil {
		t.Fatalf("expected message build to succeed: %v", err)
	}

	body := emailBody(t, string(message))
	if strings.ContainsAny(body, "\x00\x1f\x7f") {
		t.Fatalf("expected control characters to be removed, body was %q", body)
	}
	if !strings.Contains(body, "Line one\tok\r\nLine two") {
		t.Fatalf("expected tab and normalized newline to remain, body was %q", body)
	}
}

func TestNormalEmailMessageBuildsSafely(t *testing.T) {
	message, from, to, err := buildEmailMessage(
		"Ops <ops@example.com>",
		"recipient@example.com",
		"Critical incident created",
		"Incident created for asset motor-101",
	)
	if err != nil {
		t.Fatalf("expected message build to succeed: %v", err)
	}
	if from != "ops@example.com" {
		t.Fatalf("expected parsed from address, got %s", from)
	}
	if to != "recipient@example.com" {
		t.Fatalf("expected parsed recipient address, got %s", to)
	}

	raw := string(message)
	if !strings.Contains(raw, "Subject: Critical incident created\r\n") {
		t.Fatalf("expected subject header, got %q", raw)
	}
	if !strings.Contains(raw, "\r\n\r\nIncident created for asset motor-101\r\n") {
		t.Fatalf("expected body after header separator, got %q", raw)
	}
}

func TestEmailBodyCannotInjectHeaders(t *testing.T) {
	message, _, _, err := buildEmailMessage(
		"ops@example.com",
		"recipient@example.com",
		"Alert",
		"Line one\r\nBcc: attacker@example.com",
	)
	if err != nil {
		t.Fatalf("expected message build to succeed: %v", err)
	}

	raw := string(message)
	headers := emailHeaders(t, raw)
	if strings.Contains(headers, "Bcc: attacker@example.com") {
		t.Fatalf("expected body text not to appear in headers, headers were %q", headers)
	}
	if !strings.Contains(raw, "\r\n\r\nLine one\r\nBcc: attacker@example.com\r\n") {
		t.Fatalf("expected body text after header separator, got %q", raw)
	}
}

func emailHeaders(t *testing.T, message string) string {
	t.Helper()

	parts := strings.SplitN(message, "\r\n\r\n", 2)
	if len(parts) != 2 {
		t.Fatalf("expected header/body separator, got %q", message)
	}

	return parts[0]
}

func emailBody(t *testing.T, message string) string {
	t.Helper()

	parts := strings.SplitN(message, "\r\n\r\n", 2)
	if len(parts) != 2 {
		t.Fatalf("expected header/body separator, got %q", message)
	}

	return parts[1]
}
