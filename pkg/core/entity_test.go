package core

import (
	"testing"
	"time"
)

func TestNewEntity_SetsIDAndTimestampsUTC(t *testing.T) {
	before := time.Now().UTC()
	e := NewEntity[string]("user-1")
	after := time.Now().UTC()

	if e.ID != "user-1" {
		t.Errorf("ID: got %q, want %q", e.ID, "user-1")
	}
	if e.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt location: got %v, want UTC", e.CreatedAt.Location())
	}
	if e.UpdatedAt.Location() != time.UTC {
		t.Errorf("UpdatedAt location: got %v, want UTC", e.UpdatedAt.Location())
	}
	if e.CreatedAt.Before(before) || e.CreatedAt.After(after) {
		t.Errorf("CreatedAt outside [%v, %v]: got %v", before, after, e.CreatedAt)
	}
	if !e.CreatedAt.Equal(e.UpdatedAt) {
		t.Errorf("CreatedAt and UpdatedAt must be equal at construction; got %v vs %v",
			e.CreatedAt, e.UpdatedAt)
	}
}

func TestNewEntity_GenericIDInt(t *testing.T) {
	e := NewEntity[int](42)
	if e.ID != 42 {
		t.Fatalf("int ID: got %d, want 42", e.ID)
	}
}

func TestEntity_Touch_UpdatesOnlyUpdatedAt(t *testing.T) {
	e := NewEntity[string]("x")
	originalCreated := e.CreatedAt
	originalID := e.ID

	// Sleep enough that UpdatedAt strictly advances even on coarse
	// monotonic clocks (Windows has ~15ms resolution).
	time.Sleep(20 * time.Millisecond)

	e.Touch()

	if e.ID != originalID {
		t.Errorf("ID changed by Touch: got %q, want %q", e.ID, originalID)
	}
	if !e.CreatedAt.Equal(originalCreated) {
		t.Errorf("CreatedAt changed by Touch: got %v, want %v", e.CreatedAt, originalCreated)
	}
	if !e.UpdatedAt.After(originalCreated) {
		t.Errorf("UpdatedAt did not advance after Touch: original=%v, after=%v",
			originalCreated, e.UpdatedAt)
	}
	if e.UpdatedAt.Location() != time.UTC {
		t.Errorf("Touch produced non-UTC UpdatedAt: %v", e.UpdatedAt.Location())
	}
}
