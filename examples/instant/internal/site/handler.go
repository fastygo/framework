package site

import (
	"net/http"
	"strconv"
	"strings"

	instantstore "github.com/fastygo/framework/pkg/web/instant"
)

const (
	defaultCacheControl = "public, max-age=600, stale-while-revalidate=86400"
	notFoundBody        = "not found\n"
	methodNotAllowed    = "method not allowed\n"
	healthBody          = "ok\n"
)

// Handler serves one prebuilt page without runtime rendering or asset lookups.
type Handler struct {
	page          instantstore.Page
	stats         instantstore.Stats
	contentType   []string
	cacheControl  []string
	etag          []string
	contentLength []string
}

// NewHandler validates the fixed page budget at startup and keeps the hot-path
// page copy private to the handler.
func NewHandler(opts instantstore.Options) (*Handler, error) {
	store, err := instantstore.NewStore([]instantstore.Page{
		{
			Key:          pageKey,
			Body:         []byte(articleHTML),
			CacheControl: defaultCacheControl,
		},
	}, opts)
	if err != nil {
		return nil, err
	}

	page, ok := store.Get(pageKey)
	if !ok {
		return nil, http.ErrMissingFile
	}

	return &Handler{
		page:          page,
		stats:         store.Stats(),
		contentType:   []string{page.ContentType},
		cacheControl:  []string{page.CacheControl},
		etag:          []string{page.ETag},
		contentLength: []string{strconv.Itoa(len(page.Body))},
	}, nil
}

// StoreStats reports the fixed memory budget and page size configured at
// startup. It is intended for logs and tests, not per-request instrumentation.
func (h *Handler) StoreStats() instantstore.Stats {
	if h == nil {
		return instantstore.Stats{}
	}
	return h.stats
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		http.Error(w, notFoundBody, http.StatusNotFound)
		return
	}

	switch r.URL.Path {
	case pageKey, "/index.html":
		h.servePage(w, r)
	case "/healthz":
		servePlain(w, r, healthBody, http.StatusOK)
	default:
		servePlain(w, r, notFoundBody, http.StatusNotFound)
	}
}

func (h *Handler) servePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		servePlain(w, r, methodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	header := w.Header()
	header["Content-Type"] = h.contentType
	header["Cache-Control"] = h.cacheControl
	header["Etag"] = h.etag
	header["Content-Length"] = h.contentLength

	if etagMatches(r.Header.Get("If-None-Match"), h.page.ETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if r.Method == http.MethodHead {
		return
	}

	_, _ = w.Write(h.page.Body)
}

func servePlain(w http.ResponseWriter, r *http.Request, body string, status int) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		status = http.StatusMethodNotAllowed
		body = methodNotAllowed
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(body))
	}
}

func etagMatches(header string, etag string) bool {
	if header == "" || etag == "" {
		return false
	}
	for {
		candidate := header
		if comma := strings.IndexByte(header, ','); comma >= 0 {
			candidate = header[:comma]
			header = header[comma+1:]
		} else {
			header = ""
		}

		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag {
			return true
		}
		if header == "" {
			return false
		}
	}
}
