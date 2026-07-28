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
	if authErr := c.auth.Authenticate(req); authErr != nil {
		return &mail.Error{Op: op, Code: mail.CodeAuth, Err: authErr}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return &mail.Error{Op: op, Code: mail.CodeUnavailable, Err: err}
	}
	defer func() {
		_ = resp.Body.Close()
	}()

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

	_, hasMail := session.Capabilities[wire.CapMail]
	_, hasSubmission := session.Capabilities[wire.CapSubmission]
	caps := deriveCapabilities(session, hasMail)
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
		slog.Bool("threads", caps.Threads),
		slog.Bool("search", caps.Search),
		slog.Bool("push", caps.Push),
		slog.Bool("submission", hasSubmission))
	return nil
}

// deriveCapabilities maps advertised session features onto mail.Capabilities.
// Search and Threads follow CapMail (RFC 8621 Email/query + Thread/*).
// Push follows a non-empty eventSourceUrl only — never assumed.
func deriveCapabilities(session wire.Session, hasMail bool) mail.Capabilities {
	return mail.Capabilities{
		Threads: hasMail,
		Search:  hasMail,
		Push:    session.EventSourceURL != "",
	}
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
