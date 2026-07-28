package imap

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/fastygo/framework/pkg/mail"
)

const (
	idleReconnectDelay = 5 * time.Second
	defaultWatchMbox   = "INBOX"
)

// Watch implements mail.Pusher with a dedicated IMAP connection running IDLE.
// The primary Client connection stays free for ordinary commands. The returned
// channel closes when ctx is cancelled, the client is Closed, or the watcher
// exits permanently.
//
// ChangeEvent.MailboxID is the mailbox SELECT'd for IDLE (Options.WatchMailbox
// or INBOX). Events are mailbox-level only — callers re-query Messages.
func (c *Client) Watch(ctx context.Context) (<-chan mail.ChangeEvent, error) {
	const op = "imap: IDLE"
	if !c.Capabilities().Push {
		return nil, &mail.Error{Op: op, Code: mail.CodeUnsupported, Err: fmt.Errorf("server does not advertise IDLE")}
	}
	if err := ctx.Err(); err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, &mail.Error{Op: op, Code: mail.CodeUnavailable, Err: fmt.Errorf("client closed")}
	}
	if c.watchRoot == nil {
		c.mu.Unlock()
		return nil, &mail.Error{Op: op, Code: mail.CodeUnavailable, Err: fmt.Errorf("watch root missing")}
	}
	watchCtx, stop := context.WithCancel(c.watchRoot)
	c.watchWG.Add(1)
	c.mu.Unlock()

	mailbox := c.opts.WatchMailbox
	if mailbox == "" {
		mailbox = defaultWatchMbox
	}
	events := make(chan mail.ChangeEvent)
	go func() {
		defer c.watchWG.Done()
		defer stop()
		defer close(events)

		// Tie caller cancellation into the client-owned watch context.
		go func() {
			select {
			case <-ctx.Done():
				stop()
			case <-watchCtx.Done():
			}
		}()

		auditEvent(watchCtx, slog.LevelInfo, "push_started",
			slog.String("op", op),
			slog.String("mailbox", mailbox))
		defer auditEvent(context.Background(), slog.LevelInfo, "push_stopped",
			slog.String("op", op),
			slog.String("mailbox", mailbox))

		for {
			if c.isClosed() {
				return
			}
			if err := c.idleOnce(watchCtx, mailbox, events); err != nil {
				if watchCtx.Err() != nil || c.isClosed() {
					return
				}
				select {
				case <-watchCtx.Done():
					return
				case <-time.After(idleReconnectDelay):
				}
				continue
			}
			if watchCtx.Err() != nil || c.isClosed() {
				return
			}
		}
	}()
	return events, nil
}

func (c *Client) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *Client) idleOnce(ctx context.Context, mailbox string, events chan<- mail.ChangeEvent) error {
	const op = "imap: IDLE"
	ic, err := c.openIdleClient(ctx, mailbox, events)
	if err != nil {
		return err
	}
	defer func() {
		_ = ic.Logout().Wait()
		_ = ic.Close()
	}()

	idleCmd, err := ic.Idle()
	if err != nil {
		return wrapError(op, mail.CodeUnavailable, err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- idleCmd.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = idleCmd.Close()
		<-waitDone
		return ctx.Err()
	case err := <-waitDone:
		_ = idleCmd.Close()
		if err != nil {
			return wrapError(op, mail.CodeUnavailable, err)
		}
		return nil
	}
}

func (c *Client) openIdleClient(ctx context.Context, mailbox string, events chan<- mail.ChangeEvent) (*imapclient.Client, error) {
	const op = "imap: IDLE connect"
	if c.isClosed() {
		return nil, &mail.Error{Op: op, Code: mail.CodeUnavailable, Err: fmt.Errorf("client closed")}
	}
	emit := func(ev mail.ChangeEvent) {
		select {
		case events <- ev:
		case <-ctx.Done():
		}
	}
	handler := &imapclient.UnilateralDataHandler{
		Expunge: func(uint32) {
			emit(mail.ChangeEvent{MailboxID: mailbox})
		},
		Mailbox: func(data *imapclient.UnilateralDataMailbox) {
			if data == nil {
				return
			}
			emit(mail.ChangeEvent{MailboxID: mailbox})
		},
		Fetch: func(*imapclient.FetchMessageData) {
			emit(mail.ChangeEvent{MailboxID: mailbox})
		},
	}

	tlsConfig := &tls.Config{
		ServerName:         c.opts.IMAPHost,
		InsecureSkipVerify: c.opts.InsecureTLS, //nolint:gosec // explicit development-only option
	}
	imapOpts := &imapclient.Options{
		TLSConfig:             tlsConfig,
		UnilateralDataHandler: handler,
	}

	var (
		ic  *imapclient.Client
		err error
	)
	if c.opts.DialIMAP != nil {
		var conn net.Conn
		conn, err = c.opts.DialIMAP(ctx)
		if err == nil {
			ic = imapclient.New(conn, imapOpts)
		}
	} else {
		ic, err = imapclient.DialTLS(net.JoinHostPort(c.opts.IMAPHost, strconv.Itoa(c.opts.IMAPPort)), imapOpts)
	}
	if err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}
	if err := ctx.Err(); err != nil {
		_ = ic.Close()
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}
	if c.isClosed() {
		_ = ic.Close()
		return nil, &mail.Error{Op: op, Code: mail.CodeUnavailable, Err: fmt.Errorf("client closed")}
	}
	if err := ic.Login(c.opts.Username, c.opts.Password).Wait(); err != nil {
		_ = ic.Close()
		return nil, wrapError(op, mail.CodeAuth, err)
	}
	if _, err := ic.Select(mailbox, nil).Wait(); err != nil {
		_ = ic.Logout().Wait()
		_ = ic.Close()
		return nil, wrapError(op, mail.CodeNotFound, err)
	}
	return ic, nil
}

func supportsIdle(caps goimap.CapSet) bool {
	return caps.Has(goimap.CapIdle) || caps.Has(goimap.CapIMAP4rev2)
}
