package jmap_test

import (
	"context"
	"fmt"
	"log"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap"
)

// Example demonstrates connecting to a JMAP server (e.g. Stalwart) and
// listing the inbox. It is compiled but not executed: it needs a live
// server.
func Example() {
	ctx := context.Background()

	sessionURL, err := jmap.Discover("mail.example.com")
	if err != nil {
		log.Print(err)
		return
	}

	client, err := jmap.New(ctx, jmap.Options{
		SessionURL: sessionURL,
		Auth:       mail.BasicAuth{Username: "ada@example.com", Password: "app-password"},
	})
	if err != nil {
		log.Print(err)
		return
	}
	defer client.Close()

	boxes, err := client.Mailboxes(ctx)
	if err != nil {
		log.Print(err)
		return
	}
	inbox := mail.FindByRole(boxes, mail.RoleInbox)

	page, err := client.Messages(ctx, inbox.ID, mail.ListOptions{Limit: 20})
	if err != nil {
		log.Print(err)
		return
	}
	for _, msg := range page.Items {
		fmt.Println(msg.Envelope.Subject)
	}
}
