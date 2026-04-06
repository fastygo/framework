package web

import "github.com/fastygo/framework/pkg/app"

type ThemeToggleData struct {
	Label            string
	SwitchToDarkLabel string
	SwitchToLightLabel string
}

type PageData struct {
	Title            string
	Locale           string
	Active           string
	BrandName        string
	NavItems         []app.NavItem
	Theme            ThemeToggleData
	LanguageToggle   LanguageToggle
}

type LanguageToggle struct {
	Current    string
	Available  []string
	Labels     map[string]string
	SwitchTo   string
}
