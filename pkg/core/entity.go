package core

import "time"

// Entity is a generic base type for domain entities with an
// identity and audit timestamps. Embed it (not assign it) into
// concrete entities so the ID type can be any comparable, e.g.:
//
//	type User struct {
//	    core.Entity[string]
//	    Email string
//	}
type Entity[ID comparable] struct {
	// ID is the primary identifier of the entity.
	ID ID
	// CreatedAt is the UTC creation timestamp set by NewEntity.
	CreatedAt time.Time
	// UpdatedAt is the UTC modification timestamp; call Touch on
	// every mutating operation to keep it accurate.
	UpdatedAt time.Time
}

// NewEntity returns a fresh Entity[ID] with both audit timestamps
// set to time.Now().UTC() and the supplied id.
func NewEntity[ID comparable](id ID) Entity[ID] {
	now := time.Now().UTC()
	return Entity[ID]{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Touch updates UpdatedAt to time.Now().UTC(). Repositories should
// call it once per mutating operation just before persistence.
func (e *Entity[ID]) Touch() {
	e.UpdatedAt = time.Now().UTC()
}
