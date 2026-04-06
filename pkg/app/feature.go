package app

import "net/http"

type NavItem struct {
	Label string
	Path  string
	Icon  string
	Order int
}

type Feature interface {
	ID() string
	Routes(mux *http.ServeMux)
	NavItems() []NavItem
}
