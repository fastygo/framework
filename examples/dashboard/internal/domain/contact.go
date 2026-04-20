// Package domain holds the dashboard's domain types.
package domain

import (
	"errors"
	"strings"
	"time"
)

// Validation errors are stable codes, while human-readable messages come
// from the i18n layer.
var ErrContactNameRequired = errors.New("contact_name_required")
var ErrContactEmailRequired = errors.New("contact_email_required")

// Contact is a tiny CRM contact record.
type Contact struct {
	ID        string
	Name      string
	Email     string
	Company   string
	CreatedAt time.Time
}

// Validate enforces minimal invariants on a Contact.
func (c Contact) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return ErrContactNameRequired
	}
	if strings.TrimSpace(c.Email) == "" {
		return ErrContactEmailRequired
	}
	return nil
}
