package instant

import (
	"strings"
	"testing"
)

func TestNewStoreStatsAndDefaults(t *testing.T) {
	t.Parallel()

	store, err := NewStore([]Page{
		{Key: "/", Body: []byte("<h1>Home</h1>")},
		{Key: "/ru", Body: []byte("<h1>RU</h1>"), ContentType: "text/html", CacheControl: "public, max-age=30"},
	}, Options{MaxPages: 3, MaxBytes: 100})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	stats := store.Stats()
	if stats.Pages != 2 {
		t.Fatalf("Stats Pages = %d, want 2", stats.Pages)
	}
	if stats.Bytes != len("<h1>Home</h1>")+len("<h1>RU</h1>") {
		t.Fatalf("Stats Bytes = %d", stats.Bytes)
	}
	if stats.MaxPages != 3 || stats.MaxBytes != 100 {
		t.Fatalf("Stats budgets = %+v, want MaxPages=3 MaxBytes=100", stats)
	}

	page, ok := store.Get("/")
	if !ok {
		t.Fatal("expected home page")
	}
	if page.ContentType != defaultContentType {
		t.Fatalf("default ContentType = %q, want %q", page.ContentType, defaultContentType)
	}
	if page.CacheControl != defaultCacheControl {
		t.Fatalf("default CacheControl = %q, want %q", page.CacheControl, defaultCacheControl)
	}
	if page.ETag == "" {
		t.Fatal("default ETag must be set")
	}
}

func TestNewStoreRejectsBudgetOverflow(t *testing.T) {
	t.Parallel()

	_, err := NewStore([]Page{
		{Key: "a", Body: []byte("12345")},
		{Key: "b", Body: []byte("67890")},
	}, Options{MaxPages: 1})
	if err == nil || !strings.Contains(err.Error(), "page count") {
		t.Fatalf("expected page count budget error, got %v", err)
	}

	_, err = NewStore([]Page{
		{Key: "a", Body: []byte("12345")},
		{Key: "b", Body: []byte("67890")},
	}, Options{MaxBytes: 9})
	if err == nil || !strings.Contains(err.Error(), "byte budget") {
		t.Fatalf("expected byte budget error, got %v", err)
	}
}

func TestNewStoreRejectsInvalidKeys(t *testing.T) {
	t.Parallel()

	_, err := NewStore([]Page{{Body: []byte("missing")}}, Options{})
	if err == nil || !strings.Contains(err.Error(), "key is required") {
		t.Fatalf("expected missing key error, got %v", err)
	}

	_, err = NewStore([]Page{
		{Key: "same", Body: []byte("one")},
		{Key: "same", Body: []byte("two")},
	}, Options{})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate key error, got %v", err)
	}
}

func TestStoreCopiesInputAndOutputBodies(t *testing.T) {
	t.Parallel()

	body := []byte("original")
	store, err := NewStore([]Page{{Key: "/", Body: body}}, Options{})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	body[0] = 'X'
	page, ok := store.Get("/")
	if !ok {
		t.Fatal("expected page")
	}
	if string(page.Body) != "original" {
		t.Fatalf("store body changed through input slice: %q", page.Body)
	}

	page.Body[0] = 'Y'
	again, ok := store.Get("/")
	if !ok {
		t.Fatal("expected page on second get")
	}
	if string(again.Body) != "original" {
		t.Fatalf("store body changed through returned slice: %q", again.Body)
	}
}

func TestStoreNil(t *testing.T) {
	t.Parallel()

	var store *Store
	if _, ok := store.Get("/"); ok {
		t.Fatal("nil store should miss")
	}
	if got := store.Stats(); got != (Stats{}) {
		t.Fatalf("nil Stats = %+v, want zero", got)
	}
}
