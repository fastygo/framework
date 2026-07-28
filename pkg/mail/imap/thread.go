package imap

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	gomail "github.com/emersion/go-message/mail"
	"github.com/fastygo/framework/pkg/mail"
)

// threadWindowMax caps how many recent messages are scanned for client-side
// threading when THREAD=REFERENCES is unavailable. Not a global account graph.
const threadWindowMax = 500

const threadIDPrefix = "thr"

// Thread implements mail.Threader. threadID is encodeThreadID(mailbox, rootMsgID)
// as assigned on MessageSummary.ThreadID. Messages are returned oldest first.
//
// Prefer IMAP UID THREAD REFERENCES when the server advertises it; otherwise
// scan up to threadWindowMax recent messages in the mailbox and group by
// Message-ID / In-Reply-To / References headers.
func (c *Client) Thread(ctx context.Context, threadID string) ([]mail.MessageSummary, error) {
	const op = "imap: Thread"
	mailbox, rootID, err := decodeThreadID(threadID)
	if err != nil {
		return nil, wrapError(op, mail.CodeProtocol, err)
	}
	if rootID == "" {
		return nil, wrapError(op, mail.CodeProtocol, fmt.Errorf("empty thread root"))
	}

	unlock, err := c.lock(ctx, op)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := c.selectLocked(mailbox, op); err != nil {
		return nil, err
	}

	var uids []goimap.UID
	if c.threadRefs {
		uids, err = c.threadUIDsFromServerLocked(op, rootID)
		if err != nil {
			return nil, err
		}
	}
	if len(uids) == 0 {
		uids, err = c.threadUIDsClientSideLocked(op, rootID)
		if err != nil {
			return nil, err
		}
	}
	if len(uids) == 0 {
		return nil, wrapError(op, mail.CodeNotFound, fmt.Errorf("thread %q", threadID))
	}

	fetched, err := c.imap.Fetch(goimap.UIDSetNum(uids...), &goimap.FetchOptions{
		UID:          true,
		Flags:        true,
		Envelope:     true,
		RFC822Size:   true,
		InternalDate: true,
		BodyStructure: &goimap.FetchItemBodyStructure{Extended: true},
	}).Collect()
	if err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}
	byUID := make(map[goimap.UID]*imapclient.FetchMessageBuffer, len(fetched))
	for _, item := range fetched {
		byUID[item.UID] = item
	}
	out := make([]mail.MessageSummary, 0, len(uids))
	for _, uid := range uids {
		item := byUID[uid]
		if item == nil {
			continue
		}
		out = append(out, summaryFromFetch(mailbox, item))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Envelope.Date.Before(out[j].Envelope.Date)
	})
	if len(out) == 0 {
		return nil, wrapError(op, mail.CodeNotFound, fmt.Errorf("thread %q", threadID))
	}
	return out, nil
}

func (c *Client) threadUIDsFromServerLocked(op, rootID string) ([]goimap.UID, error) {
	trees, err := c.imap.UIDThread(&imapclient.ThreadOptions{
		Algorithm:      goimap.ThreadReferences,
		SearchCriteria: &goimap.SearchCriteria{},
	}).Wait()
	if err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}
	// Map UID -> whether in matching tree requires fetching Message-IDs.
	// Flatten all trees, fetch envelopes for candidates in window, then pick
	// the tree whose messages share rootID.
	allUIDs := flattenThreadUIDs(trees)
	if len(allUIDs) == 0 {
		return nil, nil
	}
	if len(allUIDs) > threadWindowMax {
		allUIDs = allUIDs[len(allUIDs)-threadWindowMax:]
	}
	headers, err := c.fetchThreadHeadersLocked(op, allUIDs)
	if err != nil {
		return nil, err
	}
	wanted := map[goimap.UID]struct{}{}
	for uid, meta := range headers {
		if threadRoot(meta.messageID, meta.inReplyTo, meta.references) == rootID || meta.messageID == rootID {
			wanted[uid] = struct{}{}
		}
	}
	// Expand: include anyone who references a wanted Message-ID (one pass closure).
	idToUID := map[string]goimap.UID{}
	for uid, meta := range headers {
		if meta.messageID != "" {
			idToUID[meta.messageID] = uid
		}
	}
	changed := true
	for changed {
		changed = false
		for uid, meta := range headers {
			if _, ok := wanted[uid]; ok {
				continue
			}
			if belongsToThread(meta, rootID, wanted, idToUID, headers) {
				wanted[uid] = struct{}{}
				changed = true
			}
		}
	}
	out := make([]goimap.UID, 0, len(wanted))
	for uid := range wanted {
		out = append(out, uid)
	}
	return out, nil
}

func (c *Client) threadUIDsClientSideLocked(op, rootID string) ([]goimap.UID, error) {
	search, err := c.imap.UIDSearch(&goimap.SearchCriteria{}, nil).Wait()
	if err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}
	uids := search.AllUIDs()
	if len(uids) == 0 {
		return nil, nil
	}
	if len(uids) > threadWindowMax {
		uids = uids[len(uids)-threadWindowMax:]
	}
	headers, err := c.fetchThreadHeadersLocked(op, uids)
	if err != nil {
		return nil, err
	}
	idToUID := map[string]goimap.UID{}
	for uid, meta := range headers {
		if meta.messageID != "" {
			idToUID[meta.messageID] = uid
		}
	}
	wanted := map[goimap.UID]struct{}{}
	for uid, meta := range headers {
		if threadRoot(meta.messageID, meta.inReplyTo, meta.references) == rootID || meta.messageID == rootID {
			wanted[uid] = struct{}{}
		}
	}
	changed := true
	for changed {
		changed = false
		for uid, meta := range headers {
			if _, ok := wanted[uid]; ok {
				continue
			}
			if belongsToThread(meta, rootID, wanted, idToUID, headers) {
				wanted[uid] = struct{}{}
				changed = true
			}
		}
	}
	out := make([]goimap.UID, 0, len(wanted))
	for _, uid := range uids {
		if _, ok := wanted[uid]; ok {
			out = append(out, uid)
		}
	}
	return out, nil
}

type threadHeader struct {
	messageID  string
	inReplyTo  []string
	references []string
}

func (c *Client) fetchThreadHeadersLocked(op string, uids []goimap.UID) (map[goimap.UID]threadHeader, error) {
	section := &goimap.FetchItemBodySection{
		Specifier:    goimap.PartSpecifierHeader,
		HeaderFields: []string{"Message-ID", "In-Reply-To", "References"},
		Peek:         true,
	}
	fetched, err := c.imap.Fetch(goimap.UIDSetNum(uids...), &goimap.FetchOptions{
		UID:         true,
		Envelope:    true,
		BodySection: []*goimap.FetchItemBodySection{section},
	}).Collect()
	if err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}
	out := make(map[goimap.UID]threadHeader, len(fetched))
	for _, item := range fetched {
		meta := threadHeader{}
		if item.Envelope != nil {
			meta.messageID = trimMessageID(item.Envelope.MessageID)
			meta.inReplyTo = trimMessageIDs(item.Envelope.InReplyTo)
		}
		raw := item.FindBodySection(section)
		if len(raw) > 0 {
			if hdr, err := gomail.CreateReader(bytes.NewReader(append(raw, "\r\n"...))); err == nil {
				if id, err := hdr.Header.MessageID(); err == nil {
					meta.messageID = trimMessageID(id)
				}
				if ids, err := hdr.Header.MsgIDList("In-Reply-To"); err == nil {
					meta.inReplyTo = trimMessageIDs(ids)
				}
				if ids, err := hdr.Header.MsgIDList("References"); err == nil {
					meta.references = trimMessageIDs(ids)
				}
				_ = hdr.Close()
			}
		}
		out[item.UID] = meta
	}
	return out, nil
}

func belongsToThread(meta threadHeader, rootID string, wanted map[goimap.UID]struct{}, idToUID map[string]goimap.UID, headers map[goimap.UID]threadHeader) bool {
	if meta.messageID == rootID {
		return true
	}
	for _, id := range meta.inReplyTo {
		if id == rootID {
			return true
		}
		if uid, ok := idToUID[id]; ok {
			if _, ok := wanted[uid]; ok {
				return true
			}
		}
	}
	for _, id := range meta.references {
		if id == rootID {
			return true
		}
		if uid, ok := idToUID[id]; ok {
			if _, ok := wanted[uid]; ok {
				return true
			}
		}
	}
	_ = headers
	return false
}

func flattenThreadUIDs(trees []imapclient.ThreadData) []goimap.UID {
	var out []goimap.UID
	var walk func(imapclient.ThreadData)
	walk = func(node imapclient.ThreadData) {
		for _, n := range node.Chain {
			out = append(out, goimap.UID(n))
		}
		for _, sub := range node.SubThreads {
			walk(sub)
		}
	}
	for _, tree := range trees {
		walk(tree)
	}
	return out
}

func encodeThreadID(mailbox, rootMsgID string) string {
	return threadIDPrefix + idSeparator + mailbox + idSeparator + rootMsgID
}

func decodeThreadID(id string) (mailbox, rootMsgID string, err error) {
	parts := strings.SplitN(id, idSeparator, 3)
	if len(parts) != 3 || parts[0] != threadIDPrefix || parts[1] == "" || parts[2] == "" {
		return "", "", fmt.Errorf("invalid IMAP thread ID")
	}
	return parts[1], parts[2], nil
}

// threadRoot picks the conversation root Message-ID: first References entry,
// else first In-Reply-To, else the message's own Message-ID.
func threadRoot(messageID string, inReplyTo, references []string) string {
	if len(references) > 0 && references[0] != "" {
		return references[0]
	}
	if len(inReplyTo) > 0 && inReplyTo[0] != "" {
		return inReplyTo[0]
	}
	return messageID
}

func supportsThreadReferences(caps goimap.CapSet) bool {
	for _, alg := range caps.ThreadAlgorithms() {
		if alg == goimap.ThreadReferences {
			return true
		}
	}
	return false
}
