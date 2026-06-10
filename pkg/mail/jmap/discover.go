package jmap

import (
	"fmt"
	"net/url"
	"strings"
)

// Discover derives the JMAP session resource URL from a server host using
// the .well-known/jmap convention (RFC 8620 §2.2). It accepts a bare host
// ("mail.example.com"), a host:port, or an https:// URL prefix and always
// produces an https URL.
//
// SRV-record autodiscovery is intentionally out of scope: webmail
// applications configure their server explicitly (see ADR 0004).
func Discover(host string) (string, error) {
	if host == "" {
		return "", fmt.Errorf("jmap: discover: empty host")
	}
	raw := host
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("jmap: discover: %w", err)
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("jmap: discover: scheme %q not allowed, use https", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("jmap: discover: no host in %q", host)
	}
	u.Path = "/.well-known/jmap"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}
