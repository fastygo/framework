package mail

// Standard message keywords shared across transports. JMAP uses these
// exact names (RFC 8621 §4.1.1); the IMAP transport maps system flags
// (\Seen, \Flagged, ...) onto them.
const (
	// FlagSeen marks a message as read.
	FlagSeen = "$seen"
	// FlagFlagged marks a message as starred / pinned.
	FlagFlagged = "$flagged"
	// FlagAnswered marks a message as replied to.
	FlagAnswered = "$answered"
	// FlagForwarded marks a message as forwarded.
	FlagForwarded = "$forwarded"
	// FlagDraft marks a message as an unsent draft.
	FlagDraft = "$draft"
)

// FlagSet is a set of message keywords. Keys are keyword names such as
// FlagSeen or arbitrary client-defined keywords.
type FlagSet map[string]bool

// NewFlagSet builds a FlagSet from keyword names.
func NewFlagSet(flags ...string) FlagSet {
	s := make(FlagSet, len(flags))
	for _, f := range flags {
		s[f] = true
	}
	return s
}

// Has reports whether the keyword is present in the set.
func (s FlagSet) Has(flag string) bool { return s[flag] }

// Names returns the keywords present in the set in unspecified order.
func (s FlagSet) Names() []string {
	out := make([]string, 0, len(s))
	for f, ok := range s {
		if ok {
			out = append(out, f)
		}
	}
	return out
}
