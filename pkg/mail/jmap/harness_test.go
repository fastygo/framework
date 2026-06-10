package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

// fakeServer simulates the JMAP endpoints of a Stalwart-style server:
// session resource, API endpoint with scripted method handlers, and
// blob download/upload endpoints.
type fakeServer struct {
	t   *testing.T
	srv *httptest.Server
	// handlers maps method name to a scripted response generator.
	handlers map[string]func(args json.RawMessage, callID string) wire.Invocation
	// requests records every parsed API request for assertions.
	requests []wire.Request
	// blobs maps blobId to content served by the download endpoint.
	blobs map[string]string
	// uploads records uploaded blob contents in order.
	uploads []string
	// requireAuth is the expected basic-auth user; empty disables check.
	user, pass string
}

func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()
	f := &fakeServer{
		t:        t,
		handlers: map[string]func(json.RawMessage, string) wire.Invocation{},
		blobs:    map[string]string{},
		user:     "ada@example.com",
		pass:     "secret",
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jmap", f.handleSession)
	mux.HandleFunc("/api", f.handleAPI)
	mux.HandleFunc("/download/", f.handleDownload)
	mux.HandleFunc("/upload/", f.handleUpload)
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not in this harness", http.StatusNotImplemented)
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeServer) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok || user != f.user || pass != f.pass {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

func (f *fakeServer) handleSession(w http.ResponseWriter, r *http.Request) {
	if !f.checkAuth(w, r) {
		return
	}
	base := f.srv.URL
	session := map[string]any{
		"capabilities": map[string]any{
			wire.CapCore: map[string]any{
				"maxSizeUpload":  50000000,
				"maxSizeRequest": 10000000,
			},
			wire.CapMail:       map[string]any{},
			wire.CapSubmission: map[string]any{},
		},
		"accounts": map[string]any{
			"a1": map[string]any{"name": "ada@example.com", "isPersonal": true},
		},
		"primaryAccounts": map[string]string{
			wire.CapMail: "a1",
		},
		"apiUrl":         base + "/api",
		"downloadUrl":    base + "/download/{accountId}/{blobId}/{name}?accept={type}",
		"uploadUrl":      base + "/upload/{accountId}/",
		"eventSourceUrl": base + "/events?types={types}&closeafter={closeafter}&ping={ping}",
		"state":          "s1",
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(session); err != nil {
		f.t.Errorf("encode session: %v", err)
	}
}

func (f *fakeServer) handleAPI(w http.ResponseWriter, r *http.Request) {
	if !f.checkAuth(w, r) {
		return
	}
	var req wire.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.requests = append(f.requests, req)

	resp := wire.Response{SessionState: "s1"}
	for _, inv := range req.MethodCalls {
		handler, ok := f.handlers[inv.Name]
		if !ok {
			resp.MethodResponses = append(resp.MethodResponses, errorInvocation("unknownMethod", inv.CallID))
			continue
		}
		resp.MethodResponses = append(resp.MethodResponses, handler(inv.Args, inv.CallID))
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		f.t.Errorf("encode response: %v", err)
	}
}

func (f *fakeServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	if !f.checkAuth(w, r) {
		return
	}
	// Path: /download/{accountId}/{blobId}/{name}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/download/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	content, ok := f.blobs[parts[1]]
	if !ok {
		http.NotFound(w, r)
		return
	}
	_, _ = io.WriteString(w, content)
}

func (f *fakeServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if !f.checkAuth(w, r) {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.uploads = append(f.uploads, string(body))
	blobID := fmt.Sprintf("upload-%d", len(f.uploads))
	f.blobs[blobID] = string(body)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(wire.UploadResponse{
		AccountID: "a1",
		BlobID:    blobID,
		Type:      r.Header.Get("Content-Type"),
		Size:      int64(len(body)),
	})
}

// respond builds a scripted method handler returning static args.
func respond(method string, args any) func(json.RawMessage, string) wire.Invocation {
	return func(_ json.RawMessage, callID string) wire.Invocation {
		raw, err := json.Marshal(args)
		if err != nil {
			panic(err)
		}
		return wire.Invocation{Name: method, Args: raw, CallID: callID}
	}
}

func errorInvocation(errType, callID string) wire.Invocation {
	raw, _ := json.Marshal(wire.MethodError{Type: errType})
	return wire.Invocation{Name: "error", Args: raw, CallID: callID}
}

// connect builds a Client against the fake server.
func (f *fakeServer) connect(t *testing.T) *Client {
	t.Helper()
	client, err := New(context.Background(), Options{
		SessionURL: f.srv.URL + "/.well-known/jmap",
		Auth:       mail.BasicAuth{Username: f.user, Password: f.pass},
		HTTPClient: f.srv.Client(),
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// fixtureEmail returns a wire-shaped email object for fixtures.
func fixtureEmail(id, threadID, mailboxID, subject string) map[string]any {
	return map[string]any{
		"id":            id,
		"threadId":      threadID,
		"mailboxIds":    map[string]bool{mailboxID: true},
		"keywords":      map[string]bool{"$seen": true},
		"size":          2048,
		"receivedAt":    "2026-06-01T10:00:00Z",
		"sentAt":        "2026-06-01T09:59:00Z",
		"messageId":     []string{id + "@example.com"},
		"from":          []map[string]string{{"name": "Ada", "email": "ada@example.com"}},
		"to":            []map[string]string{{"email": "bob@example.com"}},
		"subject":       subject,
		"preview":       "preview of " + subject,
		"hasAttachment": false,
	}
}
