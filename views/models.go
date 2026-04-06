package views

import (
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/views/partials"
)

type ThemeToggleData struct {
	Label             string
	SwitchToDarkLabel string
	SwitchToLightLabel string
}

type LayoutData struct {
	Title       string
	Locale      string
	Active      string
	BrandName   string
	NavItems    []app.NavItem
	ThemeToggle ThemeToggleData
	LanguageToggle partials.LanguageToggleData
}

type WelcomePageData struct {
	Title       string
	Description string
	ButtonLabel string
}
