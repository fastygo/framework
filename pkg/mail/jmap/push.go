package jmap

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

// maxEventBytes bounds a single EventSource line so a misbehaving server
// cannot grow the scanner buffer without limit.
const maxEventBytes = 1 << 20

// reconnectDelay is the pause before re-dialing a dropped event stream.
const reconnectDelay = 5 * time.Second

// Watch implements mail.Pusher using the JMAP EventSource endpoint
// (RFC 8620 §7.3). The returned channel closes when ctx is cancelled or
// the stream fails permanently; the single goroutine behind it is owned
// by ctx and exits with it (goleak-verified).
func (c *Client) Watch(ctx context.Context) (<-chan mail.ChangeEvent, error) {
	const op = "jmap: eventsource"

	_, _, _, eventsTmpl := c.sessionURLs()
	if eventsTmpl == "" {
		return nil, &mail.Error{Op: op, Code: mail.CodeUnsupported, Err: fmt.Errorf("session has no eventSourceUrl")}
	}
	eventsURL := expandURITemplate(eventsTmpl, map[string]string{
		"types":      "Email,Mailbox",
		"closeafter": "no",
		"ping":       "60",
	})

	events := make(chan mail.ChangeEvent)
	go func() {
		defer close(events)
		auditEvent(ctx, slog.LevelInfo, "push_started", slog.String("op", op))
		defer auditEvent(ctx, slog.LevelInfo, "push_stopped", slog.String("op", op))

		for {
			if err := c.streamEvents(ctx, eventsURL, events); err != nil {
				if ctx.Err() != nil {
					return
				}
				// Transient failure: back off, then re-dial.
				select {
				case <-ctx.Done():
					return
				case <-time.After(reconnectDelay):
				}
				continue
			}
			// Server closed the stream cleanly; re-dial immediately
			// unless we are shutting down.
			if ctx.Err() != nil {
				return
			}
		}
	}()
	return events, nil
}

// streamEvents runs one EventSource connection until it ends. It returns
// nil on clean EOF and an error on dial/read failures.
func (c *Client) streamEvents(ctx context.Context, eventsURL string, events chan<- mail.ChangeEvent) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, eventsURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	if authErr := c.auth.Authenticate(req); authErr != nil {
		return authErr
	}

	// The shared c.http carries a global timeout that would kill a
	// long-lived stream; clone it without the timeout. Lifetime is
	// still bounded by ctx through the request.
	streamClient := &http.Client{Transport: c.http.Transport}
	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("eventsource status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), maxEventBytes)

	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		case line == "":
			if len(dataLines) > 0 {
				c.dispatchEvent(ctx, strings.Join(dataLines, "\n"), events)
				dataLines = dataLines[:0]
			}
		default:
			// Comments (": ping") and other fields are ignored.
		}
	}
	return scanner.Err()
}

// dispatchEvent parses one StateChange payload and forwards change events.
func (c *Client) dispatchEvent(ctx context.Context, payload string, events chan<- mail.ChangeEvent) {
	var change wire.StateChange
	if err := json.Unmarshal([]byte(payload), &change); err != nil {
		// Unknown event payloads are skipped, not fatal.
		return
	}
	account := c.account()
	if _, ok := change.Changed[account]; !ok {
		return
	}
	select {
	case events <- mail.ChangeEvent{}:
	case <-ctx.Done():
	}
}
