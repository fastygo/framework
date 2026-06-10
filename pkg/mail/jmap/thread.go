package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

// Thread implements mail.Threader. It batches Thread/get and Email/get in
// one round-trip via a back-reference and returns messages oldest first.
func (c *Client) Thread(ctx context.Context, threadID string) ([]mail.MessageSummary, error) {
	const op = "jmap: Thread/get"

	account := c.account()
	resp, err := c.batch(ctx, op, []string{wire.CapCore, wire.CapMail},
		call{
			method: "Thread/get",
			args: map[string]any{
				"accountId": account,
				"ids":       []string{threadID},
			},
		},
		call{
			method: "Email/get",
			args: map[string]any{
				"accountId": account,
				"#ids": map[string]any{
					"resultOf": "c0",
					"name":     "Thread/get",
					"path":     "/list/*/emailIds",
				},
				"properties": summaryProperties,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	var get wire.GetResponse
	if err := c.result(op, resp, "c0", "Thread/get", &get); err != nil {
		return nil, err
	}
	var threads []struct {
		ID       string   `json:"id"`
		EmailIDs []string `json:"emailIds"`
	}
	if err := json.Unmarshal(get.List, &threads); err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}
	if len(threads) == 0 {
		return nil, &mail.Error{Op: op, Code: mail.CodeNotFound, Err: fmt.Errorf("thread %q", threadID)}
	}

	var emailGet wire.GetResponse
	if err := c.result(op, resp, "c1", "Email/get", &emailGet); err != nil {
		return nil, err
	}
	var emails []wire.Email
	if err := json.Unmarshal(emailGet.List, &emails); err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}

	out := make([]mail.MessageSummary, len(emails))
	for i := range emails {
		out[i] = toSummary(&emails[i])
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Envelope.Date.Before(out[j].Envelope.Date)
	})
	return out, nil
}
