package imap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"strings"
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message"
	gomail "github.com/emersion/go-message/mail"
	"github.com/fastygo/framework/pkg/mail"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// Messages implements mail.Client.
func (c *Client) Messages(ctx context.Context, mailboxID string, opts mail.ListOptions) (mail.Page[mail.MessageSummary], error) {
	const op = "imap: Messages"
	page := mail.Page[mail.MessageSummary]{Items: []mail.MessageSummary{}, Offset: opts.Offset}
	if opts.Offset < 0 {
		return page, wrapError(op, mail.CodeProtocol, fmt.Errorf("offset must not be negative"))
	}
	if opts.SortBy != "" && opts.SortBy != mail.SortDate {
		return page, wrapError(op, mail.CodeUnsupported, fmt.Errorf("sort field %q is not supported by IMAP F1", opts.SortBy))
	}
	limit := pageLimit(opts.Limit)

	unlock, err := c.lock(ctx, op)
	if err != nil {
		return page, err
	}
	defer unlock()
	if err := c.selectLocked(mailboxID, op); err != nil {
		return page, err
	}

	criteria := &goimap.SearchCriteria{}
	if opts.UnreadOnly {
		criteria.NotFlag = []goimap.Flag{goimap.FlagSeen}
	}
	search, err := c.imap.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return page, wrapError(op, mail.CodeUnavailable, err)
	}
	uids := search.AllUIDs()
	page.Total = int64(len(uids))
	if !opts.Ascending {
		reverseUIDs(uids)
	}

	start := opts.Offset
	if start >= int64(len(uids)) {
		return page, nil
	}
	end := start + int64(limit)
	if end > int64(len(uids)) {
		end = int64(len(uids))
	}
	selected := uids[int(start):int(end)]
	if len(selected) == 0 {
		return page, nil
	}

	fetched, err := c.imap.Fetch(goimap.UIDSetNum(selected...), &goimap.FetchOptions{
		UID:           true,
		Flags:         true,
		Envelope:      true,
		RFC822Size:    true,
		InternalDate:  true,
		BodyStructure: &goimap.FetchItemBodyStructure{Extended: true},
	}).Collect()
	if err != nil {
		return page, wrapError(op, mail.CodeUnavailable, err)
	}
	byUID := make(map[goimap.UID]*imapclient.FetchMessageBuffer, len(fetched))
	for _, item := range fetched {
		byUID[item.UID] = item
	}
	for _, uid := range selected {
		item := byUID[uid]
		if item == nil {
			continue
		}
		page.Items = append(page.Items, summaryFromFetch(mailboxID, item))
	}
	return page, nil
}

// Message implements mail.Client.
func (c *Client) Message(ctx context.Context, id string) (*mail.Message, error) {
	const op = "imap: Message"
	mailbox, uid, err := decodeMessageID(id)
	if err != nil {
		return nil, wrapError(op, mail.CodeProtocol, err)
	}
	unlock, err := c.lock(ctx, op)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := c.selectLocked(mailbox, op); err != nil {
		return nil, err
	}

	section := &goimap.FetchItemBodySection{Peek: true}
	fetched, err := c.imap.Fetch(goimap.UIDSetNum(uid), &goimap.FetchOptions{
		UID:           true,
		Flags:         true,
		Envelope:      true,
		RFC822Size:    true,
		InternalDate:  true,
		BodyStructure: &goimap.FetchItemBodyStructure{Extended: true},
		BodySection:   []*goimap.FetchItemBodySection{section},
	}).Collect()
	if err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}
	if isNotFoundFetch(fetched) {
		return nil, wrapError(op, mail.CodeNotFound, fmt.Errorf("message not found"))
	}
	item := fetched[0]
	raw := item.FindBodySection(section)
	if raw == nil {
		return nil, wrapError(op, mail.CodeProtocol, fmt.Errorf("server omitted message body"))
	}

	reader, readErr := gomail.CreateReader(bytes.NewReader(raw))
	if reader == nil {
		return nil, wrapError(op, mail.CodeProtocol, readErr)
	}
	defer reader.Close()

	result := &mail.Message{MessageSummary: summaryFromFetch(mailbox, item)}
	result.Envelope = envelopeFromHeader(&reader.Header, result.Envelope, item.InternalDate)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil && part == nil {
			return nil, wrapError(op, mail.CodeProtocol, err)
		}
		contentType := mediaType(part.Header.Get("Content-Type"))
		if _, inline := part.Header.(*gomail.InlineHeader); inline {
			body, err := io.ReadAll(part.Body)
			if err != nil {
				return nil, wrapError(op, mail.CodeProtocol, err)
			}
			switch contentType {
			case "text/plain":
				if result.TextBody == "" {
					result.TextBody = string(body)
				}
			case "text/html":
				if result.HTMLBody == "" {
					result.HTMLBody = string(body)
				}
			}
		}
	}
	result.Attachments = attachmentsFromBodyStructure(item.BodyStructure)
	result.HasAttachments = len(result.Attachments) > 0
	return result, nil
}

// Attachment implements mail.Client.
func (c *Client) Attachment(ctx context.Context, messageID, partID string) (io.ReadCloser, mail.AttachmentInfo, error) {
	const op = "imap: Attachment"
	var zero mail.AttachmentInfo
	mailbox, uid, err := decodeMessageID(messageID)
	if err != nil {
		return nil, zero, wrapError(op, mail.CodeProtocol, err)
	}
	part, err := parsePartID(partID)
	if err != nil {
		return nil, zero, wrapError(op, mail.CodeProtocol, err)
	}
	unlock, err := c.lock(ctx, op)
	if err != nil {
		return nil, zero, err
	}
	defer unlock()
	if err := c.selectLocked(mailbox, op); err != nil {
		return nil, zero, err
	}

	bodySection := &goimap.FetchItemBodySection{Part: part, Peek: true}
	mimeSection := &goimap.FetchItemBodySection{Part: part, Specifier: goimap.PartSpecifierMIME, Peek: true}
	fetched, err := c.imap.Fetch(goimap.UIDSetNum(uid), &goimap.FetchOptions{
		UID:           true,
		BodyStructure: &goimap.FetchItemBodyStructure{Extended: true},
		BodySection:   []*goimap.FetchItemBodySection{mimeSection, bodySection},
	}).Collect()
	if err != nil {
		return nil, zero, wrapError(op, mail.CodeUnavailable, err)
	}
	if isNotFoundFetch(fetched) {
		return nil, zero, wrapError(op, mail.CodeNotFound, fmt.Errorf("message not found"))
	}
	info, found := attachmentAtPart(fetched[0].BodyStructure, part)
	if !found {
		return nil, zero, wrapError(op, mail.CodeNotFound, fmt.Errorf("attachment part not found"))
	}
	header := fetched[0].FindBodySection(mimeSection)
	body := fetched[0].FindBodySection(bodySection)
	if body == nil {
		return nil, zero, wrapError(op, mail.CodeNotFound, fmt.Errorf("attachment content not found"))
	}
	raw := make([]byte, 0, len(header)+len(body)+2)
	raw = append(raw, header...)
	if len(header) > 0 && !bytes.HasSuffix(header, []byte("\r\n\r\n")) {
		raw = append(raw, []byte("\r\n")...)
	}
	raw = append(raw, body...)
	entity, err := message.Read(bytes.NewReader(raw))
	if err != nil {
		return nil, zero, wrapError(op, mail.CodeProtocol, err)
	}
	decoded, err := io.ReadAll(entity.Body)
	if err != nil {
		return nil, zero, wrapError(op, mail.CodeProtocol, err)
	}
	return io.NopCloser(bytes.NewReader(decoded)), info, nil
}

func reverseUIDs(uids []goimap.UID) {
	for i, j := 0, len(uids)-1; i < j; i, j = i+1, j-1 {
		uids[i], uids[j] = uids[j], uids[i]
	}
}

func summaryFromFetch(mailbox string, item *imapclient.FetchMessageBuffer) mail.MessageSummary {
	summary := mail.MessageSummary{
		ID:         encodeMessageID(mailbox, item.UID),
		MailboxIDs: []string{mailbox},
		Flags:      flagsFromIMAP(stringsFromIMAPFlags(item.Flags)),
		Size:       item.RFC822Size,
	}
	summary.Envelope = envelopeFromIMAP(item.Envelope, item.InternalDate)
	summary.HasAttachments = len(attachmentsFromBodyStructure(item.BodyStructure)) > 0
	if root := threadRoot(summary.Envelope.MessageID, summary.Envelope.InReplyTo, summary.Envelope.References); root != "" {
		summary.ThreadID = encodeThreadID(mailbox, root)
	}
	return summary
}

func envelopeFromIMAP(env *goimap.Envelope, fallback time.Time) mail.Envelope {
	if env == nil {
		return mail.Envelope{Date: fallback}
	}
	date := env.Date
	if date.IsZero() {
		date = fallback
	}
	return mail.Envelope{
		From:      addressesFromIMAP(env.From),
		To:        addressesFromIMAP(env.To),
		Cc:        addressesFromIMAP(env.Cc),
		Bcc:       addressesFromIMAP(env.Bcc),
		ReplyTo:   addressesFromIMAP(env.ReplyTo),
		Subject:   env.Subject,
		Date:      date,
		MessageID: trimMessageID(env.MessageID),
		InReplyTo: trimMessageIDs(env.InReplyTo),
	}
}

func addressesFromIMAP(input []goimap.Address) []mail.Address {
	out := make([]mail.Address, 0, len(input))
	for i := range input {
		if address := input[i].Addr(); address != "" {
			out = append(out, mail.Address{Name: input[i].Name, Email: address})
		}
	}
	return out
}

func envelopeFromHeader(header *gomail.Header, fallback mail.Envelope, internalDate time.Time) mail.Envelope {
	env := fallback
	env.From = addressesFromHeader(header, "From")
	env.To = addressesFromHeader(header, "To")
	env.Cc = addressesFromHeader(header, "Cc")
	env.Bcc = addressesFromHeader(header, "Bcc")
	env.ReplyTo = addressesFromHeader(header, "Reply-To")
	if subject, err := header.Subject(); err == nil {
		env.Subject = subject
	}
	if date, err := header.Date(); err == nil && !date.IsZero() {
		env.Date = date
	} else if env.Date.IsZero() {
		env.Date = internalDate
	}
	if id, err := header.MessageID(); err == nil {
		env.MessageID = trimMessageID(id)
	}
	if ids, err := header.MsgIDList("In-Reply-To"); err == nil {
		env.InReplyTo = trimMessageIDs(ids)
	}
	if ids, err := header.MsgIDList("References"); err == nil {
		env.References = trimMessageIDs(ids)
	}
	return env
}

func addressesFromHeader(header *gomail.Header, key string) []mail.Address {
	input, err := header.AddressList(key)
	if err != nil {
		return nil
	}
	out := make([]mail.Address, 0, len(input))
	for _, address := range input {
		out = append(out, mail.Address{Name: address.Name, Email: address.Address})
	}
	return out
}

func trimMessageIDs(input []string) []string {
	out := make([]string, 0, len(input))
	for _, id := range input {
		if id = trimMessageID(id); id != "" {
			out = append(out, id)
		}
	}
	return out
}

func mediaType(value string) string {
	typ, _, err := mime.ParseMediaType(value)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(strings.SplitN(value, ";", 2)[0]))
	}
	return strings.ToLower(typ)
}

func attachmentsFromBodyStructure(bs goimap.BodyStructure) []mail.AttachmentInfo {
	if bs == nil {
		return nil
	}
	var attachments []mail.AttachmentInfo
	bs.Walk(func(path []int, part goimap.BodyStructure) bool {
		single, ok := part.(*goimap.BodyStructureSinglePart)
		if !ok {
			return true
		}
		if info, ok := attachmentInfo(single, path); ok {
			attachments = append(attachments, info)
		}
		return true
	})
	return attachments
}

func attachmentAtPart(bs goimap.BodyStructure, wanted []int) (mail.AttachmentInfo, bool) {
	if bs == nil {
		return mail.AttachmentInfo{}, false
	}
	var (
		found mail.AttachmentInfo
		ok    bool
	)
	bs.Walk(func(path []int, part goimap.BodyStructure) bool {
		if !equalPart(path, wanted) {
			return true
		}
		single, singleOK := part.(*goimap.BodyStructureSinglePart)
		if !singleOK {
			return false
		}
		found, ok = attachmentInfo(single, path)
		return false
	})
	return found, ok
}

func attachmentInfo(part *goimap.BodyStructureSinglePart, path []int) (mail.AttachmentInfo, bool) {
	disposition := part.Disposition()
	filename := part.Filename()
	isAttachment := filename != ""
	inline := false
	if disposition != nil {
		isAttachment = isAttachment || strings.EqualFold(disposition.Value, "attachment") ||
			(strings.EqualFold(disposition.Value, "inline") && part.ID != "")
		inline = strings.EqualFold(disposition.Value, "inline") && part.ID != ""
	}
	if !isAttachment {
		return mail.AttachmentInfo{}, false
	}
	return mail.AttachmentInfo{
		PartID:      formatPartID(path),
		Filename:    filename,
		ContentType: part.MediaType(),
		Size:        int64(part.Size),
		Inline:      inline,
	}, true
}

func equalPart(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
