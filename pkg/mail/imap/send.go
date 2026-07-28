package imap

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/fastygo/framework/pkg/mail"
)

// Send implements mail.Client.
func (c *Client) Send(ctx context.Context, draft mail.Draft) (*mail.SendResult, error) {
	const op = "imap: Send"
	raw, messageID, err := buildMIME(draft)
	if err != nil {
		return nil, wrapError(op, mail.CodeProtocol, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}

	recipients := recipientEmails(draft)
	if err := c.sendSMTP(ctx, draft.From.Email, recipients, raw); err != nil {
		return nil, wrapError("smtp: Send", mail.CodeUnavailable, err)
	}

	unlock, err := c.lock(ctx, op)
	if err != nil {
		return nil, err
	}
	defer unlock()
	boxes, err := c.listMailboxesLocked(op)
	if err != nil {
		return nil, err
	}
	sent := mail.FindByRole(boxes, mail.RoleSent)
	if sent == nil {
		return nil, wrapError(op, mail.CodeNotFound, fmt.Errorf("Sent mailbox not found"))
	}

	appendCmd := c.imap.Append(sent.ID, int64(len(raw)), &goimap.AppendOptions{
		Flags: []goimap.Flag{goimap.FlagSeen},
		Time:  time.Now(),
	})
	if _, err := appendCmd.Write(raw); err != nil {
		_ = appendCmd.Close()
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}
	if err := appendCmd.Close(); err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}
	appendData, err := appendCmd.Wait()
	if err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}

	result := &mail.SendResult{MessageID: messageID}
	if appendData.UID != 0 {
		result.SentMessageID = encodeMessageID(sent.ID, appendData.UID)
	}
	return result, nil
}

func (c *Client) sendSMTP(ctx context.Context, from string, recipients []string, raw []byte) error {
	var (
		client *smtp.Client
		err    error
	)
	if c.opts.DialSMTP != nil {
		var conn net.Conn
		conn, err = c.opts.DialSMTP(ctx)
		if err == nil {
			client = smtp.NewClient(conn)
		}
	} else {
		tlsConfig := &tls.Config{
			ServerName:         c.opts.SMTPHost,
			InsecureSkipVerify: c.opts.InsecureTLS, //nolint:gosec // explicit development-only option
		}
		address := net.JoinHostPort(c.opts.SMTPHost, strconv.Itoa(c.opts.SMTPPort))
		client, err = smtp.DialTLS(address, tlsConfig)
	}
	if err != nil {
		return err
	}
	defer client.Close()
	if err := ctx.Err(); err != nil {
		return err
	}

	auth := sasl.NewPlainClient("", c.opts.Username, c.opts.Password)
	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.SendMail(from, recipients, bytes.NewReader(raw)); err != nil {
		return err
	}
	return client.Quit()
}

func recipientEmails(draft mail.Draft) []string {
	addresses := make([]mail.Address, 0, len(draft.To)+len(draft.Cc)+len(draft.Bcc))
	addresses = append(addresses, draft.To...)
	addresses = append(addresses, draft.Cc...)
	addresses = append(addresses, draft.Bcc...)
	out := make([]string, len(addresses))
	for i := range addresses {
		out[i] = addresses[i].Email
	}
	return out
}
