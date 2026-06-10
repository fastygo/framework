package jmap

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

// newPushServer builds a fake server whose /events endpoint streams the
// given StateChange payloads then blocks until the request context ends.
func newPushServer(t *testing.T, payloads []string) *fakeServer {
	t.Helper()
	f := newFakeServer(t)

	// Replace the harness mux entry by wrapping the whole server: the
	// httptest mux in newFakeServer registered a 501 for /events, so we
	// stand up a dedicated server here instead.
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jmap", f.handleSession)
	mux.HandleFunc("/api", f.handleAPI)
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		if !f.checkAuth(w, r) {
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("response writer is not a flusher")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, p := range payloads {
			fmt.Fprintf(w, "event: state\ndata: %s\n\n", p)
			flusher.Flush()
		}
		<-r.Context().Done()
	})
	f.srv.Close()
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func TestWatchDeliversEvents(t *testing.T) {
	payload := `{"@type":"StateChange","changed":{"a1":{"Email":"s2"}}}`
	ignored := `{"@type":"StateChange","changed":{"other-account":{"Email":"s9"}}}`
	garbage := `not-json`
	f := newPushServer(t, []string{ignored, garbage, payload})
	client := f.connect(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, err := client.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	select {
	case _, ok := <-events:
		if !ok {
			t.Fatal("events channel closed before delivering")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for change event")
	}

	// Cancelling the context must close the channel (goroutine exit).
	cancel()
	select {
	case _, ok := <-events:
		for ok {
			_, ok = <-events
		}
	case <-time.After(5 * time.Second):
		t.Fatal("events channel did not close after cancel")
	}
}

func TestWatchUnsupported(t *testing.T) {
	f := newFakeServer(t)
	client := f.connect(t)

	// Forge a session without an EventSource URL.
	client.mu.Lock()
	client.session.EventSourceURL = ""
	client.mu.Unlock()

	_, err := client.Watch(context.Background())
	if mail.CodeOf(err) != mail.CodeUnsupported {
		t.Errorf("code: got %q, want unsupported", mail.CodeOf(err))
	}
}

// Guard: keep wire import used if assertions above change.
var _ = wire.CapMail
