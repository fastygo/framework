package views

import (
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web/view"
)

type DocsListItem struct {
	Slug  string
	Title string
}

type DocsLayoutData struct {
	Title          string
	BrandName      string
	Active         string
	NavItems       []app.NavItem
	HeaderNavItems []app.NavItem
	ThemeToggle    view.ThemeToggleData
	LanguageToggle view.LanguageToggleData
}

type DocsPageData struct {
	Title       string
	HTMLContent string
}

type DocsIndexData struct {
	Title       string
	Description string
	Pages       []DocsListItem
}
