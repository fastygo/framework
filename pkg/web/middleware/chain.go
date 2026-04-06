package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler

type Chain []Middleware

func (c Chain) Then(h http.Handler) http.Handler {
	for i := len(c) - 1; i >= 0; i-- {
		h = c[i](h)
	}
	return h
}
