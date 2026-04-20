package i18n

import (
	"embed"

	"github.com/fastygo/framework/pkg/web/i18n"
)

//go:embed en/*.json ru/*.json
var fixtureFS embed.FS

// CommonFixture is the shared header/footer/locale chrome for the site.
type CommonFixture struct {
	BrandName string       `json:"brand_name"`
	Theme     ThemeFixture `json:"theme"`
	Language  LangFixture  `json:"language"`
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

// WelcomeFixture holds the welcome-page strings.
type WelcomeFixture struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ButtonLabel string `json:"button_label"`
	Kicker      string `json:"kicker"`

	ModularTitle         string `json:"modular_title"`
	ModularDescription   string `json:"modular_description"`
	BootstrapTitle       string `json:"bootstrap_title"`
	BootstrapDescription string `json:"bootstrap_description"`
	ProductionTitle      string `json:"production_title"`
	ProductionDescription string `json:"production_description"`

	GithubLabel string `json:"github_label"`
	DocsLabel   string `json:"docs_label"`
}

// Bundle is the full per-locale bundle exposed to handlers.
type Bundle struct {
	Common  CommonFixture
	Welcome WelcomeFixture
}

// Locales lists the locales shipped in this example.
var Locales = []string{"en", "ru"}

var store = i18n.New[Bundle](fixtureFS, Locales, "en", func(reader i18n.Reader, loc string) (Bundle, error) {
	common, err := i18n.DecodeSection[CommonFixture](reader, loc, "common")
	if err != nil {
		return Bundle{}, err
	}
	welcome, err := i18n.DecodeSection[WelcomeFixture](reader, loc, "welcome")
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{Common: common, Welcome: welcome}, nil
})

// Load returns the bundle for a locale, falling back to the default locale on miss.
func Load(locale string) (Bundle, error) {
	return store.Load(locale)
}
