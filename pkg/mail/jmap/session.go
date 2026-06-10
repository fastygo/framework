package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

// loadSession fetches the session resource, validates capabilities, and
// selects the account. It populates c.session, c.accountID, and c.caps.
func (c *Client) loadSession(ctx context.Context, sessionURL, wantAccount string) error {
	const op = "jmap: session"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sessionURL, nil)
	if err != nil {
		return &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}
	req.Header.Set("Accept", "application/json")
	if err := c.auth.Authenticate(req); err != nil {
		return &mail.Error{Op: op, Code: mail.CodeAuth, Err: err}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return &mail.Error{Op: op, Code: mail.CodeUnavailable, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		code := httpStatusCode(resp.StatusCode)
		if code == mail.CodeAuth {
			auditEvent(ctx, slog.LevelWarn, "auth_failed",
				slog.String("op", op),
				slog.Int("status", resp.StatusCode))
		}
		return &mail.Error{Op: op, Code: code, Err: fmt.Errorf("status %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResp))
	if err != nil {
		return &mail.Error{Op: op, Code: mail.CodeUnavailable, Err: err}
	}

	var session wire.Session
	if err := json.Unmarshal(body, &session); err != nil {
		return &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}

	if _, ok := session.Capabilities[wire.CapCore]; !ok {
		return &mail.Error{Op: op, Code: mail.CodeUnsupported, Err: fmt.Errorf("server lacks %s", wire.CapCore)}
	}
	if _, ok := session.Capabilities[wire.CapMail]; !ok {
		return &mail.Error{Op: op, Code: mail.CodeUnsupported, Err: fmt.Errorf("server lacks %s", wire.CapMail)}
	}
	if session.APIURL == "" {
		return &mail.Error{Op: op, Code: mail.CodeProtocol, Err: fmt.Errorf("session resource has no apiUrl")}
	}

	accountID := wantAccount
	if accountID == "" {
		accountID = session.PrimaryAccounts[wire.CapMail]
	}
	if accountID == "" {
		return &mail.Error{Op: op, Code: mail.CodeProtocol, Err: fmt.Errorf("no primary mail account in session")}
	}
	if _, ok := session.Accounts[accountID]; !ok {
		return &mail.Error{Op: op, Code: mail.CodeNotFound, Err: fmt.Errorf("account %q not in session", accountID)}
	}

	caps := mail.Capabilities{
		// Threads and search are part of the mandatory JMAP mail
		// capability; push depends on an advertised EventSource URL.
		Threads: true,
		Search:  true,
		Push:    session.EventSourceURL != "",
	}
	var core wire.CoreCapability
	if raw, ok := session.Capabilities[wire.CapCore]; ok {
		// Best-effort: a malformed core object only loses limits.
		_ = json.Unmarshal(raw, &core)
		caps.MaxUploadSize = core.MaxSizeUpload
		caps.MaxMessageSize = core.MaxSizeRequest
	}

	c.mu.Lock()
	c.session = session
	c.accountID = accountID
	c.mu.Unlock()
	c.caps = caps

	auditEvent(ctx, slog.LevelInfo, "session_loaded",
		slog.String("api_url", session.APIURL),
		slog.Bool("push", caps.Push))
	return nil
}

// httpStatusCode maps an HTTP status to a mail.ErrorCode.
func httpStatusCode(status int) mail.ErrorCode {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return mail.CodeAuth
	case status == http.StatusNotFound:
		return mail.CodeNotFound
	case status == http.StatusRequestEntityTooLarge:
		return mail.CodeTooLarge
	case status == http.StatusTooManyRequests:
		return mail.CodeRateLimit
	case status >= 500:
		return mail.CodeUnavailable
	default:
		return mail.CodeProtocol
	}
}
