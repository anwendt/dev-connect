package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestNewLoggerWritesJSONMetadata(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(Config{Format: "json", Level: "info", Writer: &buf})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	logger.InfoContext(context.Background(), "session prepared", slog.String("sessionId", "session-1"), slog.String("target", "dev01"))

	output := buf.String()
	assertContains(t, output, `"msg":"session prepared"`)
	assertContains(t, output, `"sessionId":"session-1"`)
	assertContains(t, output, `"target":"dev01"`)
}

func TestLoggerRedactsSensitiveAttributes(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(Config{Format: "json", Level: "debug", Writer: &buf})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	logger.DebugContext(context.Background(),
		"metadata",
		slog.String("password", "super-secret"),
		slog.String("bearerToken", "token-value"),
		slog.String("sshPrivateKey", "PRIVATE KEY"),
		slog.String("kubeconfig", "certificate-authority-data"),
		slog.String("proxyCredentials", "user:pass"),
		slog.String("target", "dev01"),
	)

	output := buf.String()
	for _, forbidden := range []string{"super-secret", "token-value", "PRIVATE KEY", "certificate-authority-data", "user:pass"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("log output leaked %q:\n%s", forbidden, output)
		}
	}
	assertContains(t, output, `"password":"[redacted]"`)
	assertContains(t, output, `"target":"dev01"`)
}

func TestNewRejectsInvalidFormatAndLevel(t *testing.T) {
	if _, err := New(Config{Format: "xml", Level: "info"}); err == nil {
		t.Fatal("invalid format accepted")
	}
	if _, err := New(Config{Format: "json", Level: "verbose"}); err == nil {
		t.Fatal("invalid level accepted")
	}
}

func assertContains(t *testing.T, data, want string) {
	t.Helper()
	if !strings.Contains(data, want) {
		t.Fatalf("data missing %q:\n%s", want, data)
	}
}
