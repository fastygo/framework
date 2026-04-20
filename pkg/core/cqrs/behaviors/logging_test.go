package behaviors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/fastygo/framework/pkg/core/cqrs"
)

type loggedRequest struct{ ID string }

func newJSONLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return logger, buf
}

func parseLogLines(t *testing.T, raw string) []map[string]any {
	t.Helper()
	var entries []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("invalid JSON log line %q: %v", line, err)
		}
		entries = append(entries, entry)
	}
	return entries
}

func TestLogging_EmitsStartAndComplete_OnSuccess(t *testing.T) {
	logger, buf := newJSONLogger()
	b := Logging{Logger: logger}

	res, err := b.Handle(context.Background(), loggedRequest{ID: "1"}, func(context.Context, any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if res != "ok" {
		t.Fatalf("result: got %v, want %q", res, "ok")
	}

	entries := parseLogLines(t, buf.String())
	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d: %v", len(entries), entries)
	}

	if entries[0]["msg"] != "cqrs:request:start" {
		t.Errorf("entry[0].msg: got %v, want cqrs:request:start", entries[0]["msg"])
	}
	if entries[1]["msg"] != "cqrs:request:complete" {
		t.Errorf("entry[1].msg: got %v, want cqrs:request:complete", entries[1]["msg"])
	}
	if !strings.Contains(entries[1]["type"].(string), "loggedRequest") {
		t.Errorf("type field should reference loggedRequest, got %v", entries[1]["type"])
	}
	if _, ok := entries[1]["duration_ms"]; !ok {
		t.Errorf("complete entry must include duration_ms field")
	}
}

func TestLogging_EmitsStartAndError_OnFailure(t *testing.T) {
	logger, buf := newJSONLogger()
	b := Logging{Logger: logger}

	want := errors.New("downstream broke")
	res, err := b.Handle(context.Background(), loggedRequest{ID: "2"}, func(context.Context, any) (any, error) {
		return nil, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("Handle must propagate the handler error, got %v", err)
	}
	if res != nil {
		t.Fatalf("result on error: got %v, want nil", res)
	}

	entries := parseLogLines(t, buf.String())
	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d: %v", len(entries), entries)
	}
	if entries[1]["msg"] != "cqrs:request:error" {
		t.Errorf("entry[1].msg: got %v, want cqrs:request:error", entries[1]["msg"])
	}
	if _, ok := entries[1]["error"]; !ok {
		t.Errorf("error entry must include the error field, got %v", entries[1])
	}
}

func TestLogging_NilLoggerFallsBackToDefault(t *testing.T) {
	// Sanity-check: a Logging value with Logger=nil must not panic.
	b := Logging{Logger: nil}
	_, err := b.Handle(context.Background(), loggedRequest{ID: "3"}, func(context.Context, any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
}

// satisfy the cqrs.HandlerFunc signature reference so go vet does not
// complain about the unused alias when handler bodies above are
// trimmed in the future.
var _ cqrs.HandlerFunc = func(context.Context, any) (any, error) { return nil, nil }
