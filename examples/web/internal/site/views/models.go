package views

import (
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web/view"
)

// LayoutData is what the layout component expects.
type LayoutData struct {
	Title          string
	Locale         string
	Active         string
	BrandName      string
	NavItems       []app.NavItem
	HeaderNavItems []app.NavItem
	ThemeToggle    view.ThemeToggleData
	LanguageToggle view.LanguageToggleData
}

// WelcomePageData is the welcome-page-specific payload.
type WelcomePageData struct {
	Title       string
	Description string
	ButtonLabel string
	Kicker      string

	ModularTitle       string
	ModularDescription string

	BootstrapTitle       string
	BootstrapDescription string

	ProductionTitle       string
	ProductionDescription string

	GithubLabel string
	DocsLabel   string
}
