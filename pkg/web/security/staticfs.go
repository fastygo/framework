package security

import (
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultImmutableExtensions = ".css|.js|.png|.jpg|.jpeg|.gif|.webp|.svg|.ico|.map|.woff|.woff2|.ttf|.eot|.otf"
)

func SecureFileServer(root string, maxAge int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := strings.TrimSpace(path.Clean(r.URL.Path))
		if cleanPath == "." {
			cleanPath = "/"
		}
		if strings.HasPrefix(cleanPath, "/") {
			cleanPath = cleanPath[1:]
		}
		if strings.Contains(cleanPath, "..") {
			http.NotFound(w, r)
			return
		}
		if hasDotSegment(cleanPath) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		requestPath := "/" + cleanPath
		file, err := http.Dir(root).Open(cleanPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		info, err := file.Stat()
		if err != nil || info.IsDir() {
			_ = file.Close()
			http.NotFound(w, r)
			return
		}

		etag := generateFileETag(info.ModTime(), info.Size())
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", cacheControlValue(cleanPath, maxAge))

		if r.Header.Get("If-None-Match") == etag {
			_ = file.Close()
			w.WriteHeader(http.StatusNotModified)
			return
		}

		_, _ = file.Seek(0, 0)
		http.ServeContent(w, r, filepath.ToSlash(requestPath), info.ModTime(), file)
		_ = file.Close()
	})
}

func hasDotSegment(rawPath string) bool {
	for _, segment := range strings.Split(rawPath, "/") {
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}
	return false
}

func generateFileETag(modTime time.Time, size int64) string {
	return fmt.Sprintf("\"%x-%x\"", modTime.UnixNano(), size)
}

func cacheControlValue(rawPath string, maxAge int) string {
	ext := strings.ToLower(filepath.Ext(rawPath))
	if strings.Contains(defaultImmutableExtensions, ext) {
		if maxAge <= 0 {
			maxAge = 86400
		}
		return fmt.Sprintf("public, max-age=%d, immutable", maxAge)
	}
	return "public, max-age=60"
}
