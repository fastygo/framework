package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed en/*.json ru/*.json
var fixtureFS embed.FS

type Localized struct {
	Common  CommonFixture  `json:"common"`
	Welcome WelcomeFixture `json:"welcome"`
}

type CommonFixture struct {
	BrandName string       `json:"brand_name"`
	Nav       []NavFixture `json:"nav"`
	Theme     ThemeFixture `json:"theme"`
	Language  LangFixture  `json:"language"`
}

type NavFixture struct {
	Label string `json:"label"`
	Path  string `json:"path"`
	Icon  string `json:"icon"`
	Order int    `json:"order"`
}

type ThemeFixture struct {
	Label            string `json:"label"`
	SwitchToDarkLabel string `json:"switch_to_dark_label"`
	SwitchToLightLabel string `json:"switch_to_light_label"`
}

type LangFixture struct {
	Label          string            `json:"label"`
	CurrentLabel   string            `json:"current_label"`
	NextLabel      string            `json:"next_label"`
	NextLocale     string            `json:"next_locale"`
	Available      []string          `json:"available"`
	LocaleLabels   map[string]string `json:"locale_labels"`
}

type WelcomeFixture struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ButtonLabel string `json:"button_label"`
}

var supportedLocales = []string{"en", "ru"}

func Locales() []string {
	return append([]string{}, supportedLocales...)
}

func Decode[T any](locale string, section string) (T, error) {
	var zero T
	if len(locale) == 0 {
		locale = "en"
	}

	path := fmt.Sprintf("%s/%s.json", locale, section)
	raw, err := fixtureFS.ReadFile(path)
	if err != nil {
		return zero, err
	}

	var decoded T
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return zero, err
	}

	return decoded, nil
}

func Load(locale string) (Localized, error) {
	common, err := Decode[CommonFixture](normalizeLocale(locale), "common")
	if err != nil {
		return Localized{}, err
	}

	welcome, err := Decode[WelcomeFixture](normalizeLocale(locale), "welcome")
	if err != nil {
		return Localized{}, err
	}

	return Localized{
		Common:  common,
		Welcome: welcome,
	}, nil
}

func normalizeLocale(locale string) string {
	if locale == "ru" || locale == "en" {
		return locale
	}
	return "en"
}
