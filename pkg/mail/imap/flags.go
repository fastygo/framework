package imap

import "github.com/fastygo/framework/pkg/mail"

// IMAP system flag names (RFC 3501) mapped to mail keywords.
const (
	imapFlagSeen     = `\Seen`
	imapFlagFlagged  = `\Flagged`
	imapFlagAnswered = `\Answered`
	imapFlagDraft    = `\Draft`
	// Forwarded has no universal system flag; many servers use $Forwarded.
	imapFlagForwarded = `$Forwarded`
)

var mailToIMAP = map[string]string{
	mail.FlagSeen:      imapFlagSeen,
	mail.FlagFlagged:   imapFlagFlagged,
	mail.FlagAnswered:  imapFlagAnswered,
	mail.FlagDraft:     imapFlagDraft,
	mail.FlagForwarded: imapFlagForwarded,
}

var imapToMail = map[string]string{
	imapFlagSeen:      mail.FlagSeen,
	imapFlagFlagged:   mail.FlagFlagged,
	imapFlagAnswered:  mail.FlagAnswered,
	imapFlagDraft:     mail.FlagDraft,
	imapFlagForwarded: mail.FlagForwarded,
	// Case variants some servers emit:
	`$forwarded`: mail.FlagForwarded,
	`$seen`:      mail.FlagSeen,
	`$flagged`:   mail.FlagFlagged,
	`$answered`:  mail.FlagAnswered,
	`$draft`:     mail.FlagDraft,
}

// flagsToIMAP converts a mail.FlagSet into IMAP flag strings for STORE.
func flagsToIMAP(set mail.FlagSet) []string {
	if set == nil {
		return nil
	}
	out := make([]string, 0, len(set))
	for name, on := range set {
		if !on {
			continue
		}
		if mapped, ok := mailToIMAP[name]; ok {
			out = append(out, mapped)
			continue
		}
		out = append(out, name)
	}
	return out
}

// flagsFromIMAP converts IMAP FETCH FLAGS into a mail.FlagSet.
func flagsFromIMAP(flags []string) mail.FlagSet {
	set := mail.FlagSet{}
	for _, f := range flags {
		if mapped, ok := imapToMail[f]; ok {
			set[mapped] = true
			continue
		}
		set[f] = true
	}
	return set
}
