package site

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	instantstore "github.com/fastygo/framework/pkg/web/instant"
)

func TestHandlerServesPrebuiltHTML(t *testing.T) {
	handler, err := NewHandler(instantstore.Options{MaxPages: 1, MaxBytes: 64 * 1024})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q", contentType)
	}
	if recorder.Header().Get("ETag") == "" {
		t.Fatal("expected ETag header")
	}
	if got := recorder.Header().Get("Content-Length"); got != strconv.Itoa(len(recorder.Body.Bytes())) {
		t.Fatalf("content-length = %q, want body length", got)
	}
	body := recorder.Body.String()
	for _, forbidden := range []string{"<script", "src=", `href="/static`, "stylesheet"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("body contains forbidden external asset marker %q", forbidden)
		}
	}
}

func TestHandlerHonorsETag(t *testing.T) {
	handler, err := NewHandler(instantstore.Options{MaxPages: 1, MaxBytes: 64 * 1024})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))

	second := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("If-None-Match", first.Header().Get("ETag"))

	handler.ServeHTTP(second, request)

	if second.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", second.Code, http.StatusNotModified)
	}
	if second.Body.Len() != 0 {
		t.Fatalf("304 body length = %d, want 0", second.Body.Len())
	}
}

func TestHandlerBudgetIsExplicit(t *testing.T) {
	handler, err := NewHandler(instantstore.Options{MaxPages: 1, MaxBytes: 64 * 1024})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	stats := handler.StoreStats()
	if stats.Pages != 1 {
		t.Fatalf("stats.Pages = %d, want 1", stats.Pages)
	}
	if stats.Bytes <= 0 {
		t.Fatalf("stats.Bytes = %d, want > 0", stats.Bytes)
	}
	if stats.MaxPages != 1 || stats.MaxBytes != 64*1024 {
		t.Fatalf("stats limits = (%d, %d), want (1, %d)", stats.MaxPages, stats.MaxBytes, 64*1024)
	}

	_, err = NewHandler(instantstore.Options{MaxPages: 1, MaxBytes: 1})
	if err == nil {
		t.Fatal("NewHandler() error = nil, want byte budget failure")
	}
}

type discardResponseWriter struct {
	header http.Header
	status int
}

func (w *discardResponseWriter) Header() http.Header {
	return w.header
}

func (w *discardResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *discardResponseWriter) WriteHeader(status int) {
	w.status = status
}

func BenchmarkHandlerRoot(b *testing.B) {
	handler, err := NewHandler(instantstore.Options{MaxPages: 1, MaxBytes: 64 * 1024})
	if err != nil {
		b.Fatalf("NewHandler() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	writer := &discardResponseWriter{header: make(http.Header, 8)}

	b.ReportAllocs()
	b.SetBytes(int64(handler.StoreStats().Bytes))
	for range b.N {
		writer.status = 0
		handler.ServeHTTP(writer, request)
	}
}

func BenchmarkHandlerNotModified(b *testing.B) {
	handler, err := NewHandler(instantstore.Options{MaxPages: 1, MaxBytes: 64 * 1024})
	if err != nil {
		b.Fatalf("NewHandler() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("If-None-Match", handler.page.ETag)
	writer := &discardResponseWriter{header: make(http.Header, 8)}

	b.ReportAllocs()
	for range b.N {
		writer.status = 0
		handler.ServeHTTP(writer, request)
	}
}
