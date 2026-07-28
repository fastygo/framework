package imap_test

import (
	"context"
	"testing"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/imap"
)

func TestNewValidatesOptionsBeforeDial(t *testing.T) {
	tests := []struct {
		name string
		opts imap.Options
	}{
		{name: "host", opts: imap.Options{Username: "user", Password: "secret"}},
		{name: "username", opts: imap.Options{IMAPHost: "localhost", Password: "secret"}},
		{name: "password", opts: imap.Options{IMAPHost: "localhost", Username: "user"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := imap.New(context.Background(), tt.opts)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if mail.CodeOf(err) != mail.CodeProtocol {
				t.Errorf("code: got %q, want protocol", mail.CodeOf(err))
			}
		})
	}
}
