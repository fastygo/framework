package views

import (
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web"
)

type DocsListItem struct {
	Slug  string
	Title string
}

type DocsLayoutData struct {
	Title       string
	BrandName   string
	Active      string
	NavItems    []app.NavItem
	ThemeToggle web.ThemeToggleData
}

type DocsPageData struct {
	Title       string
	HTMLContent string
}

type DocsIndexData struct {
	Pages []DocsListItem
}
