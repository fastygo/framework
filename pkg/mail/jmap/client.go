// Package jmap implements the pkg/mail Client contract for JMAP servers
// (RFC 8620 core, RFC 8621 mail), with Stalwart as the reference backend.
//
// The implementation is stdlib-only. Protocol wire shapes live in the
// internal/wire subpackage and never leak into the public API.
//
// A Client is safe for concurrent use. The only background goroutine is
// the optional EventSource push stream started by Watch; it is owned by
// the caller's context.
package jmap

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

// defaultMaxResponseBytes bounds JMAP API response bodies. Large data
// (attachments) flows through the separate download endpoint as a stream,
// so 16 MiB is generous for method responses.
const defaultMaxResponseBytes = 16 << 20

// defaultMaxBodyBytes caps fetched message body values (per Email/get
// request) so a pathological message cannot exhaust memory.
const defaultMaxBodyBytes = 1 << 20

// defaultTimeout is applied to the internal http.Client when the caller
// does not supply one.
const defaultTimeout = 30 * time.Second

// Options configures a Client. SessionURL and Auth are required.
type Options struct {
	// SessionURL is the JMAP session resource URL, e.g.
	// "https://mail.example.com/.well-known/jmap". Use Discover to
	// derive it from a bare host.
	SessionURL string
	// Auth supplies credentials for every request. Required.
	Auth mail.Authenticator
	// HTTPClient overrides the transport. Nil installs a client with a
	// 30-second timeout. Supply your own to tune pooling or tracing.
	HTTPClient *http.Client
	// AccountID selects a specific account from the session resource.
	// Empty selects the primary mail account.
	AccountID string
	// MaxResponseBytes bounds API response bodies. Zero means 16 MiB.
	MaxResponseBytes int64
	// MaxBodyBytes caps fetched message body values per message. Zero
	// means 1 MiB.
	MaxBodyBytes int64
	// InsecureTLS disables TLS certificate verification. Development
	// only: enabling it emits a Warn-level mail.audit event. It is
	// ignored when HTTPClient is supplied.
	InsecureTLS bool
}

// Client implements mail.Client, mail.Threader, mail.Pusher, and
// mail.Searcher against a JMAP server.
type Client struct {
	http    *http.Client
	auth    mail.Authenticator
	maxResp int64
	maxBody int64
	caps    mail.Capabilities

	mu        sync.RWMutex
	session   wire.Session
	accountID string
}

// Interface conformance is part of the package contract.
var (
	_ mail.Client   = (*Client)(nil)
	_ mail.Threader = (*Client)(nil)
	_ mail.Pusher   = (*Client)(nil)
	_ mail.Searcher = (*Client)(nil)
)

// auditEvent mirrors the pkg/auth audit pattern: structured slog events
// under the "mail.audit" message with stable keys, Warn for
// security-relevant anomalies.
func auditEvent(ctx context.Context, level slog.Level, event string, attrs ...slog.Attr) {
	all := make([]slog.Attr, 0, len(attrs)+1)
	all = append(all, slog.String("event", event))
	all = append(all, attrs...)
	slog.Default().LogAttrs(ctx, level, "mail.audit", all...)
}

// New connects to the JMAP server: it fetches the session resource,
// verifies the core and mail capabilities, and selects the account.
func New(ctx context.Context, opts Options) (*Client, error) {
	if opts.SessionURL == "" {
		return nil, &mail.Error{Op: "jmap: connect", Code: mail.CodeProtocol, Err: fmt.Errorf("sessionURL is required")}
	}
	if opts.Auth == nil {
		return nil, &mail.Error{Op: "jmap: connect", Code: mail.CodeAuth, Err: fmt.Errorf("auth is required")}
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		transport := http.DefaultTransport
		if opts.InsecureTLS {
			auditEvent(ctx, slog.LevelWarn, "insecure_tls_enabled",
				slog.String("session_url", opts.SessionURL))
			t := http.DefaultTransport.(*http.Transport).Clone()
			t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // explicit dev-only opt-in, audited above
			transport = t
		}
		httpClient = &http.Client{Timeout: defaultTimeout, Transport: transport}
	}

	c := &Client{
		http:    httpClient,
		auth:    opts.Auth,
		maxResp: opts.MaxResponseBytes,
		maxBody: opts.MaxBodyBytes,
	}
	if c.maxResp <= 0 {
		c.maxResp = defaultMaxResponseBytes
	}
	if c.maxBody <= 0 {
		c.maxBody = defaultMaxBodyBytes
	}

	if err := c.loadSession(ctx, opts.SessionURL, opts.AccountID); err != nil {
		return nil, err
	}
	return c, nil
}

// Capabilities implements mail.Client.
func (c *Client) Capabilities() mail.Capabilities {
	return c.caps
}

// Close implements mail.Client. The JMAP transport is stateless HTTP;
// Close only drops idle connections.
func (c *Client) Close() error {
	c.http.CloseIdleConnections()
	return nil
}

// account returns the selected account ID.
func (c *Client) account() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accountID
}

// sessionURLs returns the API, download, upload, and event-source URLs
// from the current session snapshot.
func (c *Client) sessionURLs() (api, download, upload, events string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.session.APIURL, c.session.DownloadURL, c.session.UploadURL, c.session.EventSourceURL
}
