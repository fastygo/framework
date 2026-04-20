package i18n

import (
	"embed"

	"github.com/fastygo/framework/pkg/web/i18n"
)

//go:embed en/*.json ru/*.json
var fixtureFS embed.FS

type ThemeFixture struct {
	Label              string `json:"label"`
	SwitchToDarkLabel  string `json:"switch_to_dark_label"`
	SwitchToLightLabel string `json:"switch_to_light_label"`
}

type LanguageFixture struct {
	Label        string            `json:"label"`
	CurrentLabel string            `json:"current_label"`
	NextLabel    string            `json:"next_label"`
	NextLocale   string            `json:"next_locale"`
	Available    []string          `json:"available"`
	LocaleLabels map[string]string `json:"locale_labels"`
}

type LandingFixture struct {
	BrandName   string           `json:"brand_name"`
	Tagline     string           `json:"tagline"`
	Title       string           `json:"title"`
	Subtitle    string           `json:"subtitle"`
	PrimaryCTA  string           `json:"primary_cta"`
	PrimaryHref string           `json:"primary_href"`
	FooterText  string           `json:"footer_text"`
	Features    []FeatureFixture `json:"features"`
	Locale      string           `json:"locale"`
	Theme       ThemeFixture     `json:"theme"`
	Language    LanguageFixture  `json:"language"`
}

type FeatureFixture struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type Bundle struct {
	Landing LandingFixture
}

var store = i18n.New[Bundle](fixtureFS, []string{"en", "ru"}, "en", func(reader i18n.Reader, loc string) (Bundle, error) {
	landing, err := i18n.DecodeSection[LandingFixture](reader, loc, "landing")
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{Landing: landing}, nil
})

func Load(locale string) (Bundle, error) {
	return store.Load(locale)
}
