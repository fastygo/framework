package views

import (
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web/view"
)

type DocsLayoutData struct {
	Title          string
	BrandName      string
	Active         string
	NavItems       []app.NavItem
	HeaderNavItems []app.NavItem
	ThemeToggle    view.ThemeToggleData
	LanguageToggle view.LanguageToggleData
}
