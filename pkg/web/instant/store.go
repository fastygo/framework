// Package instant stores fixed, prebuilt HTTP page snapshots.
//
// Unlike pkg/cache, this package has no TTL, no cleanup path, and no
// background goroutine. Build the store once at startup from a known set of
// pages, then serve those immutable bytes directly on the request path.
package instant

import (
	"fmt"
	"hash/fnv"
)

const (
	defaultContentType  = "text/html; charset=utf-8"
	defaultCacheControl = "public, max-age=60, stale-while-revalidate=86400"
)

// Page is a prebuilt response body addressed by Key.
type Page struct {
	Key          string
	Body         []byte
	ContentType  string
	CacheControl string
	ETag         string
}

// Options caps the fixed store at construction time. Zero values are
// unlimited, suitable for tests and tiny applications.
type Options struct {
	MaxPages int
	MaxBytes int
}

// Stats reports the fixed store size and configured budgets.
type Stats struct {
	Pages    int
	Bytes    int
	MaxPages int
	MaxBytes int
}

// Store is an immutable lookup table of prebuilt pages.
type Store struct {
	pages map[string]Page
	stats Stats
}

// NewStore validates pages, copies their bodies, applies default response
// metadata, and enforces Options before returning an immutable Store.
func NewStore(pages []Page, opts Options) (*Store, error) {
	if opts.MaxPages > 0 && len(pages) > opts.MaxPages {
		return nil, fmt.Errorf("instant: page count %d exceeds max %d", len(pages), opts.MaxPages)
	}

	store := &Store{
		pages: make(map[string]Page, len(pages)),
		stats: Stats{
			MaxPages: positive(opts.MaxPages),
			MaxBytes: positive(opts.MaxBytes),
		},
	}
	for _, page := range pages {
		normalized, err := normalizePage(page)
		if err != nil {
			return nil, err
		}
		if _, exists := store.pages[normalized.Key]; exists {
			return nil, fmt.Errorf("instant: duplicate page key %q", normalized.Key)
		}
		nextBytes := store.stats.Bytes + len(normalized.Body)
		if opts.MaxBytes > 0 && nextBytes > opts.MaxBytes {
			return nil, fmt.Errorf("instant: byte budget %d exceeds max %d", nextBytes, opts.MaxBytes)
		}
		store.pages[normalized.Key] = normalized
		store.stats.Pages++
		store.stats.Bytes = nextBytes
	}
	return store, nil
}

// Get returns a copy of the page for key. The returned Body can be mutated by
// the caller without modifying the store.
func (s *Store) Get(key string) (Page, bool) {
	if s == nil {
		return Page{}, false
	}
	page, ok := s.pages[key]
	if !ok {
		return Page{}, false
	}
	page.Body = append([]byte(nil), page.Body...)
	return page, true
}

// Stats returns the store's fixed size and configured budgets.
func (s *Store) Stats() Stats {
	if s == nil {
		return Stats{}
	}
	return s.stats
}

func normalizePage(page Page) (Page, error) {
	if page.Key == "" {
		return Page{}, fmt.Errorf("instant: page key is required")
	}
	page.Body = append([]byte(nil), page.Body...)
	if page.ContentType == "" {
		page.ContentType = defaultContentType
	}
	if page.CacheControl == "" {
		page.CacheControl = defaultCacheControl
	}
	if page.ETag == "" {
		page.ETag = bodyETag(page.Body)
	}
	return page, nil
}

func bodyETag(body []byte) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write(body)
	return fmt.Sprintf("\"%x\"", hasher.Sum64())
}

func positive(value int) int {
	if value > 0 {
		return value
	}
	return 0
}
