package middleware

import "net/http"

// Middleware is the canonical net/http middleware adapter shape:
// a function that wraps one http.Handler in another.
type Middleware func(http.Handler) http.Handler

// Chain is an ordered slice of Middleware applied left-to-right.
// The first element wraps the handler outermost (sees the request first).
type Chain []Middleware

// Then applies the chain to h and returns the wrapped handler. A
// zero-length chain returns h unchanged.
func (c Chain) Then(h http.Handler) http.Handler {
	for i := len(c) - 1; i >= 0; i-- {
		h = c[i](h)
	}
	return h
}
