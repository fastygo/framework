package core

import "time"

type Entity[ID comparable] struct {
	ID        ID
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewEntity[ID comparable](id ID) Entity[ID] {
	now := time.Now().UTC()
	return Entity[ID]{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (e *Entity[ID]) Touch() {
	e.UpdatedAt = time.Now().UTC()
}
