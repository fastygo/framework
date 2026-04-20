package views

import (
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web/view"
)

type LayoutData struct {
	Title          string
	Lang           string
	Active         string
	BrandName      string
	NavItems       []app.NavItem
	HeaderNavItems []app.NavItem
	ThemeToggle    view.ThemeToggleData
	LanguageToggle view.LanguageToggleData
}

type PostListItem struct {
	Slug    string
	Title   string
	Summary string
}

type PostListData struct {
	Title       string
	Description string
	Posts       []PostListItem
}

type PostPageData struct {
	Title       string
	HTMLContent string
}
