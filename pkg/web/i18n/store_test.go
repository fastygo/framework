package i18n

import (
	"testing"
	"testing/fstest"
)

type bundle struct {
	Title string `json:"title"`
}

func TestStoreLoadFallback(t *testing.T) {
	fsys := fstest.MapFS{
		"en/page.json": &fstest.MapFile{Data: []byte(`{"title": "hello"}`)},
		"ru/page.json": &fstest.MapFile{Data: []byte(`{"title": "привет"}`)},
	}

	store := New[bundle](fsys, []string{"en", "ru"}, "en", func(r Reader, loc string) (bundle, error) {
		return DecodeSection[bundle](r, loc, "page")
	})

	got, err := store.Load("ru")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "привет" {
		t.Fatalf("expected ru bundle, got %q", got.Title)
	}

	got, err = store.Load("de")
	if err != nil {
		t.Fatalf("unexpected error for fallback: %v", err)
	}
	if got.Title != "hello" {
		t.Fatalf("expected fallback en, got %q", got.Title)
	}
}
