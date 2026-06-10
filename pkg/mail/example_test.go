package mail_test

import (
	"fmt"

	"github.com/fastygo/framework/pkg/mail"
)

// ExampleRedactEmail shows how transports and applications keep addresses
// out of structured logs.
func ExampleRedactEmail() {
	fmt.Println(mail.RedactEmail("ada@example.com"))
	fmt.Println(mail.RedactEmail("not-an-address"))
	// Output:
	// a***@example.com
	// ***
}

// ExampleFindByRole locates the special-use Trash mailbox so a "move to
// trash" action does not depend on localized folder names.
func ExampleFindByRole() {
	boxes := []mail.Mailbox{
		{ID: "m1", Name: "Inbox", Role: mail.RoleInbox},
		{ID: "m2", Name: "Korb", Role: mail.RoleTrash},
	}
	trash := mail.FindByRole(boxes, mail.RoleTrash)
	fmt.Println(trash.ID)
	// Output:
	// m2
}
