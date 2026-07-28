// Package imap implements pkg/mail.Client over IMAP (RFC 3501 / 9051)
// plus SMTP submission for Send.
//
// Framework F1 provides the base mail.Client (slice A). F2 adds Searcher.
// F3 adds Pusher via a dedicated IDLE connection. F4 adds Threader
// (THREAD=REFERENCES when available, else client-side header window).
// Consumer Gmail/Outlook OAuth is out of scope.
package imap

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-smtp"
	"github.com/fastygo/framework/pkg/mail"
)

const (
	defaultIMAPPort = 993
	defaultSMTPPort = 465
)

// Options configures an IMAP+SMTP client. Authentication uses username and
// password with IMAP LOGIN and SMTP AUTH PLAIN.
type Options struct {
	// IMAPHost is the IMAP server host (required), e.g. "mail.example.com".
	IMAPHost string
	// IMAPPort is the IMAP port. Zero defaults to 993 (IMAPS).
	IMAPPort int
	// SMTPHost is the SMTP submission host. Empty defaults to IMAPHost.
	SMTPHost string
	// SMTPPort is the SMTP submission port. Zero defaults to 465.
	SMTPPort int
	// Username is the account login (usually the email address). Required.
	Username string
	// Password is the account or app password. Required. Treat as PII.
	Password string
	// InsecureTLS disables TLS certificate verification. Development only;
	// enabling it emits a Warn-level mail.audit event.
	InsecureTLS bool
	// DialIMAP overrides implicit-TLS IMAP dialing. It is primarily useful for
	// tests and may return a plaintext connection.
	DialIMAP func(ctx context.Context) (net.Conn, error)
	// DialSMTP overrides implicit-TLS SMTP dialing. It is primarily useful for
	// tests and may return a plaintext connection.
	DialSMTP func(ctx context.Context) (net.Conn, error)
	// WatchMailbox is SELECT'd on the dedicated IDLE connection used by Watch.
	// Empty defaults to "INBOX".
	WatchMailbox string
}

func (o Options) validate() error {
	if o.IMAPHost == "" {
		return fmt.Errorf("IMAPHost is required")
	}
	if o.Username == "" {
		return fmt.Errorf("Username is required")
	}
	if o.Password == "" {
		return fmt.Errorf("Password is required")
	}
	return nil
}

func (o Options) withDefaults() Options {
	if o.IMAPPort == 0 {
		o.IMAPPort = defaultIMAPPort
	}
	if o.SMTPHost == "" {
		o.SMTPHost = o.IMAPHost
	}
	if o.SMTPPort == 0 {
		o.SMTPPort = defaultSMTPPort
	}
	return o
}

// Client implements mail.Client over a single IMAP connection and per-send
// SMTP connections. Every IMAP operation is serialized.
type Client struct {
	opts        Options
	imap        *imapclient.Client
	mu          sync.Mutex
	caps        mail.Capabilities
	threadRefs  bool
	closed      bool
	watchRoot   context.Context
	watchCancel context.CancelFunc
	watchWG     sync.WaitGroup
}

var (
	_ mail.Client   = (*Client)(nil)
	_ mail.Searcher = (*Client)(nil)
	_ mail.Pusher   = (*Client)(nil)
	_ mail.Threader = (*Client)(nil)
)

// New validates options, connects to IMAP, and authenticates the account.
func New(ctx context.Context, opts Options) (*Client, error) {
	if err := opts.validate(); err != nil {
		return nil, &mail.Error{Op: "imap: connect", Code: mail.CodeProtocol, Err: err}
	}
	opts = opts.withDefaults()

	tlsConfig := &tls.Config{
		ServerName:         opts.IMAPHost,
		InsecureSkipVerify: opts.InsecureTLS, //nolint:gosec // explicit development-only option
	}
	if opts.InsecureTLS {
		auditEvent(ctx, slog.LevelWarn, "insecure_tls_enabled",
			slog.String("imap_host", opts.IMAPHost),
			slog.String("smtp_host", opts.SMTPHost))
	}

	imapOpts := &imapclient.Options{TLSConfig: tlsConfig}
	var (
		ic  *imapclient.Client
		err error
	)
	if opts.DialIMAP != nil {
		var conn net.Conn
		conn, err = opts.DialIMAP(ctx)
		if err == nil {
			ic = imapclient.New(conn, imapOpts)
		}
	} else {
		ic, err = imapclient.DialTLS(net.JoinHostPort(opts.IMAPHost, strconv.Itoa(opts.IMAPPort)), imapOpts)
	}
	if err != nil {
		return nil, wrapError("imap: connect", mail.CodeUnavailable, err)
	}
	if err := ctx.Err(); err != nil {
		_ = ic.Close()
		return nil, wrapError("imap: connect", mail.CodeUnavailable, err)
	}
	if err := ic.Login(opts.Username, opts.Password).Wait(); err != nil {
		_ = ic.Close()
		return nil, wrapError("imap: login", mail.CodeAuth, err)
	}

	// Fetch capabilities now so go-imap can select MOVE/UIDPLUS behavior.
	serverCaps := ic.Caps()
	watchRoot, watchCancel := context.WithCancel(context.Background())
	return &Client{
		opts:        opts,
		imap:        ic,
		threadRefs:  supportsThreadReferences(serverCaps),
		watchRoot:   watchRoot,
		watchCancel: watchCancel,
		caps: mail.Capabilities{
			Search:  true,
			Push:    supportsIdle(serverCaps),
			Threads: true, // client-side always; THREAD=REFERENCES when threadRefs
		},
	}, nil
}

func auditEvent(ctx context.Context, level slog.Level, event string, attrs ...slog.Attr) {
	all := make([]slog.Attr, 0, len(attrs)+1)
	all = append(all, slog.String("event", event))
	all = append(all, attrs...)
	slog.Default().LogAttrs(ctx, level, "mail.audit", all...)
}

func wrapError(op string, fallback mail.ErrorCode, err error) error {
	if err == nil {
		return nil
	}
	var existing *mail.Error
	if errors.As(err, &existing) {
		return err
	}
	code := fallback
	var imapErr *goimap.Error
	if errors.As(err, &imapErr) {
		switch {
		case imapErr.Code == goimap.ResponseCodeNonExistent:
			code = mail.CodeNotFound
		case imapErr.Type == goimap.StatusResponseTypeBad:
			code = mail.CodeProtocol
		}
	}
	var smtpErr *smtp.SMTPError
	if errors.As(err, &smtpErr) && (smtpErr.Code == 530 || smtpErr.Code == 534 || smtpErr.Code == 535) {
		code = mail.CodeAuth
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		code = mail.CodeUnavailable
	}
	return &mail.Error{Op: op, Code: code, Err: err}
}

func (c *Client) lock(ctx context.Context, op string) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, wrapError(op, mail.CodeUnavailable, net.ErrClosed)
	}
	if err := ctx.Err(); err != nil {
		c.mu.Unlock()
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}
	return c.mu.Unlock, nil
}

func (c *Client) selectLocked(mailbox, op string) error {
	if mailbox == "" {
		return wrapError(op, mail.CodeProtocol, fmt.Errorf("mailbox ID is required"))
	}
	_, err := c.imap.Select(mailbox, nil).Wait()
	if err != nil {
		return wrapError(op, mail.CodeNotFound, err)
	}
	return nil
}

func imapFlags(flags []string) []goimap.Flag {
	out := make([]goimap.Flag, len(flags))
	for i, flag := range flags {
		out[i] = goimap.Flag(flag)
	}
	return out
}

func stringsFromIMAPFlags(flags []goimap.Flag) []string {
	out := make([]string, len(flags))
	for i, flag := range flags {
		out[i] = string(flag)
	}
	return out
}

func isNotFoundFetch(messages []*imapclient.FetchMessageBuffer) bool {
	return len(messages) == 0 || messages[0].UID == 0
}

func trimMessageID(id string) string {
	return strings.Trim(strings.TrimSpace(id), "<>")
}

// Capabilities implements mail.Client.
func (c *Client) Capabilities() mail.Capabilities {
	return c.caps
}

// Close implements mail.Client.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	cancel := c.watchCancel
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	done := make(chan struct{})
	go func() {
		c.watchWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		auditEvent(context.Background(), slog.LevelWarn, "watch_close_timeout")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	logoutErr := c.imap.Logout().Wait()
	closeErr := c.imap.Close()
	if logoutErr != nil {
		return wrapError("imap: close", mail.CodeUnavailable, logoutErr)
	}
	if closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
		return wrapError("imap: close", mail.CodeUnavailable, closeErr)
	}
	return nil
}
