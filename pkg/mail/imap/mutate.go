package imap

import (
	"context"
	"fmt"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/fastygo/framework/pkg/mail"
)

// SetFlags implements mail.Client.
func (c *Client) SetFlags(ctx context.Context, ids []string, add, remove mail.FlagSet) error {
	const op = "imap: SetFlags"
	groups, err := groupMessageIDs(ids)
	if err != nil {
		return wrapError(op, mail.CodeProtocol, err)
	}
	unlock, err := c.lock(ctx, op)
	if err != nil {
		return err
	}
	defer unlock()

	addFlags := imapFlags(flagsToIMAP(add))
	removeFlags := imapFlags(flagsToIMAP(remove))
	for mailbox, uids := range groups {
		if err := c.selectLocked(mailbox, op); err != nil {
			return err
		}
		set := goimap.UIDSetNum(uids...)
		if len(addFlags) > 0 {
			if err := c.imap.Store(set, &goimap.StoreFlags{
				Op:     goimap.StoreFlagsAdd,
				Silent: true,
				Flags:  addFlags,
			}, nil).Close(); err != nil {
				return wrapError(op, mail.CodeUnavailable, err)
			}
		}
		if len(removeFlags) > 0 {
			if err := c.imap.Store(set, &goimap.StoreFlags{
				Op:     goimap.StoreFlagsDel,
				Silent: true,
				Flags:  removeFlags,
			}, nil).Close(); err != nil {
				return wrapError(op, mail.CodeUnavailable, err)
			}
		}
	}
	return nil
}

// Move implements mail.Client.
func (c *Client) Move(ctx context.Context, ids []string, destMailboxID string) error {
	const op = "imap: Move"
	if destMailboxID == "" {
		return wrapError(op, mail.CodeProtocol, fmt.Errorf("destination mailbox ID is required"))
	}
	groups, err := groupMessageIDs(ids)
	if err != nil {
		return wrapError(op, mail.CodeProtocol, err)
	}
	unlock, err := c.lock(ctx, op)
	if err != nil {
		return err
	}
	defer unlock()

	for mailbox, uids := range groups {
		if err := c.selectLocked(mailbox, op); err != nil {
			return err
		}
		if _, err := c.imap.Move(goimap.UIDSetNum(uids...), destMailboxID).Wait(); err != nil {
			return wrapError(op, mail.CodeUnavailable, err)
		}
	}
	return nil
}

// Delete implements mail.Client.
func (c *Client) Delete(ctx context.Context, ids []string) error {
	const op = "imap: Delete"
	groups, err := groupMessageIDs(ids)
	if err != nil {
		return wrapError(op, mail.CodeProtocol, err)
	}
	unlock, err := c.lock(ctx, op)
	if err != nil {
		return err
	}
	defer unlock()

	for mailbox, uids := range groups {
		if err := c.selectLocked(mailbox, op); err != nil {
			return err
		}
		set := goimap.UIDSetNum(uids...)
		if err := c.imap.Store(set, &goimap.StoreFlags{
			Op:     goimap.StoreFlagsAdd,
			Silent: true,
			Flags:  []goimap.Flag{goimap.FlagDeleted},
		}, nil).Close(); err != nil {
			return wrapError(op, mail.CodeUnavailable, err)
		}
		if c.imap.Caps().Has(goimap.CapUIDPlus) || c.imap.Caps().Has(goimap.CapIMAP4rev2) {
			err = c.imap.UIDExpunge(set).Close()
		} else {
			err = c.imap.Expunge().Close()
		}
		if err != nil {
			return wrapError(op, mail.CodeUnavailable, err)
		}
	}
	return nil
}

func groupMessageIDs(ids []string) (map[string][]goimap.UID, error) {
	groups := make(map[string][]goimap.UID)
	for _, id := range ids {
		mailbox, uid, err := decodeMessageID(id)
		if err != nil {
			return nil, err
		}
		groups[mailbox] = append(groups[mailbox], uid)
	}
	return groups, nil
}
