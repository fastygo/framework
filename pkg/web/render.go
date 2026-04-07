package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"

	"github.com/fastygo/framework/pkg/cache"
	"github.com/a-h/templ"
)

func Render(ctx context.Context, w http.ResponseWriter, component templ.Component) error {
	buf := &bytes.Buffer{}
	if err := component.Render(ctx, buf); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := w.Write(buf.Bytes())
	return err
}

func CachedRender(ctx context.Context, w http.ResponseWriter, r *http.Request, htmlCache *cache.Cache[[]byte], key string, component templ.Component) error {
	if htmlCache == nil || key == "" {
		return Render(ctx, w, component)
	}

	if cached, ok := htmlCache.Get(key); ok {
		etag := htmlETag(cached)
		w.Header().Set("ETag", etag)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return nil
		}
		_, err := w.Write(cached)
		return err
	}

	buf := &bytes.Buffer{}
	if err := component.Render(ctx, buf); err != nil {
		return err
	}
	body := append([]byte(nil), buf.Bytes()...)
	htmlCache.Set(key, body)

	etag := htmlETag(body)
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}

	_, err := w.Write(body)
	return err
}

func htmlETag(value []byte) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write(value)
	sum := hasher.Sum64()
	return fmt.Sprintf("\"%x\"", sum)
}

func WriteJSON(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	return enc.Encode(payload)
}
