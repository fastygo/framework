package mail

// Capabilities reports what the connected server supports, so consumers
// can adapt UI and behavior without probing. Transports populate it during
// connection; values are static for the lifetime of a Client.
type Capabilities struct {
	// Threads reports whether the transport groups messages into
	// conversations (the Client also implements Threader).
	Threads bool
	// Push reports whether the transport delivers change events (the
	// Client also implements Pusher).
	Push bool
	// Search reports whether the transport supports server-side search
	// (the Client also implements Searcher).
	Search bool
	// MaxUploadSize is the largest attachment upload in bytes the
	// server accepts; 0 when unknown.
	MaxUploadSize int64
	// MaxMessageSize is the largest total outgoing message in bytes;
	// 0 when unknown.
	MaxMessageSize int64
}
