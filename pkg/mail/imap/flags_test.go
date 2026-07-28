package imap

import (
	"testing"

	"github.com/fastygo/framework/pkg/mail"
)

func TestFlagsRoundTrip(t *testing.T) {
	in := mail.NewFlagSet(mail.FlagSeen, mail.FlagFlagged, mail.FlagDraft)
	imapFlags := flagsToIMAP(in)
	out := flagsFromIMAP(imapFlags)
	for _, want := range []string{mail.FlagSeen, mail.FlagFlagged, mail.FlagDraft} {
		if !out.Has(want) {
			t.Errorf("missing %s after round-trip; imap=%v out=%v", want, imapFlags, out.Names())
		}
	}
}

func TestFlagsFromIMAPSystemNames(t *testing.T) {
	got := flagsFromIMAP([]string{`\Seen`, `\Answered`, `$Forwarded`})
	if !got.Has(mail.FlagSeen) || !got.Has(mail.FlagAnswered) || !got.Has(mail.FlagForwarded) {
		t.Fatalf("got %v", got.Names())
	}
}
