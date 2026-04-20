package i18n

import (
	"embed"

	"github.com/fastygo/framework/pkg/web/i18n"
)

//go:embed en/*.json ru/*.json
var fixtureFS embed.FS

type CommonFixture struct {
	BrandName        string            `json:"brand_name"`
	Theme            ThemeFixture      `json:"theme"`
	Language         LangFixture       `json:"language"`
	Pages            map[string]string `json:"pages"`
	LocaleName       map[string]string `json:"locale_name"`
	IndexTitle       string            `json:"index_title"`
	IndexDescription string            `json:"index_description"`
	HeaderNavItems   []NavLinkFixture  `json:"header_nav_items"`
}

type NavLinkFixture struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

type ThemeFixture struct {
	Label              string `json:"label"`
	SwitchToDarkLabel  string `json:"switch_to_dark_label"`
	SwitchToLightLabel string `json:"switch_to_light_label"`
}

type LangFixture struct {
	Label        string            `json:"label"`
	CurrentLabel string            `json:"current_label"`
	NextLabel    string            `json:"next_label"`
	NextLocale   string            `json:"next_locale"`
	Available    []string          `json:"available"`
	LocaleLabels map[string]string `json:"locale_labels"`
}

type Bundle struct {
	Common CommonFixture
}

var Locales = []string{"en", "ru"}

var store = i18n.New[Bundle](fixtureFS, Locales, "en", func(reader i18n.Reader, loc string) (Bundle, error) {
	common, err := i18n.DecodeSection[CommonFixture](reader, loc, "common")
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{Common: common}, nil
})

func Load(locale string) (Bundle, error) { return store.Load(locale) }
