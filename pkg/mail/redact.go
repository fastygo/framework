package mail

import "strings"

// RedactEmail masks an email address for logging: it keeps the first
// character of the local part and the domain, e.g. "a***@example.com".
// Empty or malformed input redacts to "***".
//
// Transports use it so audit events never carry full addresses; consuming
// applications should do the same in their own logs.
func RedactEmail(email string) string {
	at := strings.LastIndexByte(email, '@')
	if at <= 0 || at == len(email)-1 {
		return "***"
	}
	return email[:1] + "***@" + email[at+1:]
}

// RedactSubject replaces a message subject with a length marker so log
// lines can prove a subject existed without leaking content.
func RedactSubject(subject string) string {
	if subject == "" {
		return "(empty)"
	}
	return "(redacted)"
}
