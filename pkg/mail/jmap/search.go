package jmap

import (
	"context"

	"github.com/fastygo/framework/pkg/mail"
)

// Search implements mail.Searcher over Email/query filter conditions
// (RFC 8621 §4.4.1).
func (c *Client) Search(ctx context.Context, query mail.SearchQuery, opts mail.ListOptions) (mail.Page[mail.MessageSummary], error) {
	var conditions []any
	if query.Text != "" {
		conditions = append(conditions, map[string]any{"text": query.Text})
	}
	if query.From != "" {
		conditions = append(conditions, map[string]any{"from": query.From})
	}
	if query.To != "" {
		conditions = append(conditions, map[string]any{"to": query.To})
	}
	if query.Subject != "" {
		conditions = append(conditions, map[string]any{"subject": query.Subject})
	}
	if query.MailboxID != "" {
		conditions = append(conditions, map[string]any{"inMailbox": query.MailboxID})
	}
	if query.HasAttachment {
		conditions = append(conditions, map[string]any{"hasAttachment": true})
	}
	if !query.After.IsZero() {
		conditions = append(conditions, map[string]any{"after": query.After.UTC().Format("2006-01-02T15:04:05Z")})
	}
	if !query.Before.IsZero() {
		conditions = append(conditions, map[string]any{"before": query.Before.UTC().Format("2006-01-02T15:04:05Z")})
	}

	var filter map[string]any
	switch len(conditions) {
	case 0:
		filter = map[string]any{}
	case 1:
		filter = conditions[0].(map[string]any)
	default:
		filter = map[string]any{"operator": "AND", "conditions": conditions}
	}

	return c.queryMessages(ctx, "jmap: Email/query (search)", filter, opts)
}
