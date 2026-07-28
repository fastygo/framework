package mailtest_test

import (
	"testing"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/mailtest"
)

func TestRoleMailboxes(t *testing.T) {
	boxes := mailtest.RoleMailboxes()
	if mail.FindByRole(boxes, mail.RoleInbox) == nil {
		t.Fatal("missing inbox")
	}
	if mail.FindByRole(boxes, mail.RoleDrafts) == nil {
		t.Fatal("missing drafts")
	}
	if mail.FindByRole(boxes, mail.RoleSent) == nil {
		t.Fatal("missing sent")
	}
	if mail.FindByRole(boxes, mail.RoleTrash) == nil {
		t.Fatal("missing trash")
	}
}

func TestSampleSummary(t *testing.T) {
	s := mailtest.SampleSummary("mb-inbox")
	if s.ID == "" || s.Envelope.Subject == "" {
		t.Fatalf("incomplete summary: %+v", s)
	}
	if s.Envelope.Date != mailtest.FixedTime() {
		t.Errorf("Envelope.Date: got %v", s.Envelope.Date)
	}
}

func TestFlagFixtures(t *testing.T) {
	if !mailtest.SeenFlagSet().Has(mail.FlagSeen) {
		t.Fatal("SeenFlagSet")
	}
	if !mailtest.DraftFlagSet().Has(mail.FlagDraft) {
		t.Fatal("DraftFlagSet")
	}
}
