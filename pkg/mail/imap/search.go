package imap

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/fastygo/framework/pkg/mail"
)

// Search implements mail.Searcher using IMAP UID SEARCH. Empty SearchQuery
// fields are ignored and combine with AND. When MailboxID is empty every
// selectable mailbox is searched and results are merged (newest first by
// default). HasAttachment is applied after FETCH using BODYSTRUCTURE.
func (c *Client) Search(ctx context.Context, query mail.SearchQuery, opts mail.ListOptions) (mail.Page[mail.MessageSummary], error) {
	const op = "imap: Search"
	page := mail.Page[mail.MessageSummary]{Items: []mail.MessageSummary{}, Offset: opts.Offset}
	if opts.Offset < 0 {
		return page, wrapError(op, mail.CodeProtocol, fmt.Errorf("offset must not be negative"))
	}
	if opts.SortBy != "" && opts.SortBy != mail.SortDate {
		return page, wrapError(op, mail.CodeUnsupported, fmt.Errorf("sort field %q is not supported by IMAP search", opts.SortBy))
	}
	limit := pageLimit(opts.Limit)

	unlock, err := c.lock(ctx, op)
	if err != nil {
		return page, err
	}
	defer unlock()

	mailboxes, err := c.searchMailboxesLocked(op, query.MailboxID)
	if err != nil {
		return page, err
	}
	criteria := searchCriteriaFromQuery(query)

	var hits []uidHit
	for _, mailbox := range mailboxes {
		if err := c.selectLocked(mailbox, op); err != nil {
			return page, err
		}
		data, err := c.imap.UIDSearch(criteria, nil).Wait()
		if err != nil {
			return page, wrapError(op, mail.CodeUnavailable, err)
		}
		for _, uid := range data.AllUIDs() {
			hits = append(hits, uidHit{mailbox: mailbox, uid: uid})
		}
	}

	if len(hits) == 0 {
		return page, nil
	}

	// Fetch enough metadata to sort by date and optionally filter attachments.
	fetched, err := c.fetchHitsLocked(op, hits, query.HasAttachment)
	if err != nil {
		return page, err
	}
	if query.HasAttachment {
		filtered := make([]uidHit, 0, len(fetched))
		kept := make(map[string]*imapclient.FetchMessageBuffer, len(fetched))
		for _, hit := range hits {
			item := fetched[encodeMessageID(hit.mailbox, hit.uid)]
			if item == nil {
				continue
			}
			if len(attachmentsFromBodyStructure(item.BodyStructure)) == 0 {
				continue
			}
			filtered = append(filtered, hit)
			kept[encodeMessageID(hit.mailbox, hit.uid)] = item
		}
		hits = filtered
		fetched = kept
	}

	sortHits(hits, fetched, opts.Ascending)
	page.Total = int64(len(hits))

	start := opts.Offset
	if start >= int64(len(hits)) {
		return page, nil
	}
	end := start + int64(limit)
	if end > int64(len(hits)) {
		end = int64(len(hits))
	}
	selected := hits[int(start):int(end)]
	for _, hit := range selected {
		item := fetched[encodeMessageID(hit.mailbox, hit.uid)]
		if item == nil {
			continue
		}
		page.Items = append(page.Items, summaryFromFetch(hit.mailbox, item))
	}
	return page, nil
}

type uidHit struct {
	mailbox string
	uid     goimap.UID
}

func searchCriteriaFromQuery(query mail.SearchQuery) *goimap.SearchCriteria {
	criteria := &goimap.SearchCriteria{}
	if query.Text != "" {
		criteria.Text = []string{query.Text}
	}
	if query.From != "" {
		criteria.Header = append(criteria.Header, goimap.SearchCriteriaHeaderField{
			Key: "FROM", Value: query.From,
		})
	}
	if query.To != "" {
		// IMAP has no combined To/Cc key; search To and rely on callers
		// refining with Text when Cc matters.
		criteria.Header = append(criteria.Header, goimap.SearchCriteriaHeaderField{
			Key: "TO", Value: query.To,
		})
	}
	if query.Subject != "" {
		criteria.Header = append(criteria.Header, goimap.SearchCriteriaHeaderField{
			Key: "SUBJECT", Value: query.Subject,
		})
	}
	// SearchQuery After/Before are exclusive at the Go API; IMAP SINCE is
	// inclusive of the calendar day and BEFORE is exclusive of that day.
	if !query.After.IsZero() {
		criteria.Since = truncateDayUTC(query.After).AddDate(0, 0, 1)
	}
	if !query.Before.IsZero() {
		criteria.Before = truncateDayUTC(query.Before)
	}
	return criteria
}

func truncateDayUTC(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func (c *Client) searchMailboxesLocked(op, mailboxID string) ([]string, error) {
	if mailboxID != "" {
		return []string{mailboxID}, nil
	}
	items, err := c.imap.List("", "*", &goimap.ListOptions{ReturnSpecialUse: true}).Collect()
	if err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if mailboxUnselectable(item.Attrs) {
			continue
		}
		out = append(out, item.Mailbox)
	}
	return out, nil
}

func mailboxUnselectable(attrs []goimap.MailboxAttr) bool {
	for _, attr := range attrs {
		if strings.EqualFold(string(attr), string(goimap.MailboxAttrNoSelect)) {
			return true
		}
	}
	return false
}

func (c *Client) fetchHitsLocked(op string, hits []uidHit, needStructure bool) (map[string]*imapclient.FetchMessageBuffer, error) {
	byMailbox := make(map[string][]goimap.UID)
	for _, hit := range hits {
		byMailbox[hit.mailbox] = append(byMailbox[hit.mailbox], hit.uid)
	}
	out := make(map[string]*imapclient.FetchMessageBuffer, len(hits))
	options := &goimap.FetchOptions{
		UID:          true,
		Flags:        true,
		Envelope:     true,
		RFC822Size:   true,
		InternalDate: true,
	}
	if needStructure {
		options.BodyStructure = &goimap.FetchItemBodyStructure{Extended: true}
	}
	for mailbox, uids := range byMailbox {
		if err := c.selectLocked(mailbox, op); err != nil {
			return nil, err
		}
		fetched, err := c.imap.Fetch(goimap.UIDSetNum(uids...), options).Collect()
		if err != nil {
			return nil, wrapError(op, mail.CodeUnavailable, err)
		}
		for _, item := range fetched {
			out[encodeMessageID(mailbox, item.UID)] = item
		}
	}
	return out, nil
}

func sortHits(hits []uidHit, fetched map[string]*imapclient.FetchMessageBuffer, ascending bool) {
	type scored struct {
		hit  uidHit
		date time.Time
	}
	items := make([]scored, len(hits))
	for i, hit := range hits {
		date := time.Time{}
		if item := fetched[encodeMessageID(hit.mailbox, hit.uid)]; item != nil {
			date = item.InternalDate
			if item.Envelope != nil && !item.Envelope.Date.IsZero() {
				date = item.Envelope.Date
			}
		}
		items[i] = scored{hit: hit, date: date}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].date.Equal(items[j].date) {
			if ascending {
				return items[i].hit.uid < items[j].hit.uid
			}
			return items[i].hit.uid > items[j].hit.uid
		}
		if ascending {
			return items[i].date.Before(items[j].date)
		}
		return items[i].date.After(items[j].date)
	})
	for i := range items {
		hits[i] = items[i].hit
	}
}

func pageLimit(limit int) int {
	if limit <= 0 {
		return defaultPageSize
	}
	if limit > maxPageSize {
		return maxPageSize
	}
	return limit
}
