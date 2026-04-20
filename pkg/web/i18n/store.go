// Package i18n provides a tiny generic loader for embedded JSON locale
// fixtures laid out as `<locale>/<section>.json` inside an fs.FS.
//
// Applications typically embed their messages with `//go:embed en/*.json` and
// then expose them through a Store backed by their own typed structs:
//
//	//go:embed en/*.json ru/*.json
//	var messagesFS embed.FS
//
//	type Bundle struct {
//	    Common  CommonFixture  `json:"common"`
//	    Welcome WelcomeFixture `json:"welcome"`
//	}
//
//	store := i18n.New[Bundle](messagesFS, []string{"en", "ru"}, "en", LoadBundle)
package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"sync"

	"github.com/fastygo/framework/pkg/web/locale"
)

// LoaderFunc materialises a typed bundle from a locale's JSON files.
type LoaderFunc[T any] func(reader Reader, loc string) (T, error)

// Reader exposes the section-level JSON loading API to LoaderFunc closures
// without leaking the underlying fs.FS implementation.
type Reader interface {
	Section(locale string, section string, dst any) error
}

// Store memoises one bundle per locale.
type Store[T any] struct {
	fsys     fs.FS
	locales  []string
	fallback string
	loader   LoaderFunc[T]

	once    sync.Once
	bundles map[string]T
	loadErr error
}

// New constructs a Store. The fallback locale must be present in locales.
func New[T any](fsys fs.FS, locales []string, fallback string, loader LoaderFunc[T]) *Store[T] {
	normalized := locale.Normalize(locales...)
	if len(normalized) == 0 {
		normalized = []string{"en"}
	}
	fb := locale.Normalize(fallback)
	fallbackLocale := ""
	if len(fb) > 0 {
		fallbackLocale = fb[0]
	}
	if !locale.Contains(normalized, fallbackLocale) {
		fallbackLocale = normalized[0]
	}
	return &Store[T]{
		fsys:     fsys,
		locales:  normalized,
		fallback: fallbackLocale,
		loader:   loader,
	}
}

// Locales returns the locales the store was configured with.
func (s *Store[T]) Locales() []string {
	out := make([]string, len(s.locales))
	copy(out, s.locales)
	sort.Strings(out)
	return out
}

// Fallback is the locale used when a requested one is missing.
func (s *Store[T]) Fallback() string {
	return s.fallback
}

// Load returns the bundle for the requested locale, falling back when missing.
// On the first call all locales are loaded eagerly so subsequent reads are
// lock-free.
func (s *Store[T]) Load(loc string) (T, error) {
	s.once.Do(s.preload)

	var zero T
	if s.loadErr != nil {
		return zero, s.loadErr
	}

	target := normalizeLocale(loc)
	if bundle, ok := s.bundles[target]; ok {
		return bundle, nil
	}
	if bundle, ok := s.bundles[s.fallback]; ok {
		return bundle, nil
	}
	return zero, fmt.Errorf("i18n: locale %q not loaded", loc)
}

func (s *Store[T]) preload() {
	s.bundles = make(map[string]T, len(s.locales))
	reader := embeddedReader{fsys: s.fsys}

	for _, loc := range s.locales {
		bundle, err := s.loader(reader, loc)
		if err != nil {
			s.loadErr = fmt.Errorf("i18n: load locale %q: %w", loc, err)
			return
		}
		s.bundles[loc] = bundle
	}
}

type embeddedReader struct {
	fsys fs.FS
}

func (r embeddedReader) Section(loc string, section string, dst any) error {
	if dst == nil {
		return fmt.Errorf("i18n: nil destination")
	}
	loc = normalizeLocale(loc)
	if loc == "" {
		return fmt.Errorf("i18n: empty locale")
	}
	path := fmt.Sprintf("%s/%s.json", loc, section)
	raw, err := fs.ReadFile(r.fsys, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dst)
}

// DecodeSection is a convenience helper for loaders that want to decode a
// single JSON section into a typed value.
func DecodeSection[T any](reader Reader, loc string, section string) (T, error) {
	var dst T
	if err := reader.Section(loc, section, &dst); err != nil {
		return dst, err
	}
	return dst, nil
}

func normalizeLocale(value string) string {
	normalized := locale.Normalize(value)
	if len(normalized) == 0 {
		return ""
	}
	return normalized[0]
}
