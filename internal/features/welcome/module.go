package welcome

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/core/cqrs"
	"github.com/fastygo/framework/pkg/web"
	"github.com/fastygo/framework/views"
	"github.com/fastygo/framework/views/partials"
)

type Module struct {
	dispatcher     *cqrs.Dispatcher
	navItems       []app.NavItem
	defaultLocale  string
	availableLocales []string
}

func New(dispatcher *cqrs.Dispatcher, defaultLocale string, availableLocales []string) *Module {
	return &Module{
		dispatcher:      dispatcher,
		availableLocales: availableLocales,
		defaultLocale:    defaultLocale,
		navItems: []app.NavItem{
			{
				Label: "Welcome",
				Path:  "/",
				Icon:  "home",
				Order: 0,
			},
		},
	}
}

func (m *Module) ID() string {
	return "welcome"
}

func (m *Module) NavItems() []app.NavItem {
	return append([]app.NavItem{}, m.navItems...)
}

func (m *Module) SetNavItems(items []app.NavItem) {
	m.navItems = append([]app.NavItem{}, items...)
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/", m.handleWelcome)
}

func (m *Module) handleWelcome(w http.ResponseWriter, r *http.Request) {
	locale := resolveLocale(r, m.defaultLocale, m.availableLocales)
	result, err := cqrs.DispatchQuery[WelcomeQuery, WelcomeQueryResult](r.Context(), m.dispatcher, WelcomeQuery{Locale: locale})
	if err != nil {
		web.HandleError(w, err)
		return
	}

	common := result.Layout.Common
	welcome := result.Layout.Welcome

	theme := views.ThemeToggleData{
		Label:             common.Theme.Label,
		SwitchToDarkLabel: common.Theme.SwitchToDarkLabel,
		SwitchToLightLabel: common.Theme.SwitchToLightLabel,
	}

	nextLocale := common.Language.NextLocale
	nextLabel := common.Language.NextLabel
	if nextLocale == "" {
		for _, available := range common.Language.Available {
			if available != locale {
				nextLocale = available
				break
			}
		}
	}

	if nextLabel == "" && nextLocale != "" && len(common.Language.LocaleLabels) > 0 {
		nextLabel = common.Language.LocaleLabels[nextLocale]
	}

	language := partials.LanguageToggleData{
		Label:            common.Language.Label,
		CurrentLocale:    locale,
		CurrentLabel:     common.Language.CurrentLabel,
		NextLocale:       nextLocale,
		NextLabel:        nextLabel,
		DefaultLocale:    m.defaultLocale,
		AvailableLocales: common.Language.Available,
	}

	layout := views.LayoutData{
		Title:          "Framework",
		Locale:         locale,
		Active:         "/",
		BrandName:      common.BrandName,
		NavItems:       m.navItems,
		ThemeToggle:    theme,
		LanguageToggle: language,
	}

	page := views.WelcomePageData{
		Title:       welcome.Title,
		Description: welcome.Description,
		ButtonLabel: welcome.ButtonLabel,
	}

	renderErr := web.Render(r.Context(), w, views.Layout(layout, templ.NopComponent, views.WelcomePage(page)))
	if renderErr != nil {
		web.HandleError(w, renderErr)
	}
}

func resolveLocale(r *http.Request, defaultLocale string, allowedLocales []string) string {
	locale := r.URL.Query().Get("lang")
	if locale != "" && len(locale) >= 2 {
		locale = locale[:2]
	}
	if locale == "" {
		locale = defaultLocale
	}
	if !containsLocale(allowedLocales, locale) {
		locale = defaultLocale
	}
	return locale
}

func containsLocale(allowedLocales []string, locale string) bool {
	for _, allowed := range allowedLocales {
		if allowed == locale {
			return true
		}
	}
	return false
}
