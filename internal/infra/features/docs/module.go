package docs

import (
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	appdocs "github.com/fastygo/framework/internal/application/docs"
	docsi18n "github.com/fastygo/framework/internal/site/docs/web/i18n"
	"github.com/fastygo/framework/internal/site/docs/web/views"
	"github.com/fastygo/framework/internal/site/web/views/partials"
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/cache"
	"github.com/fastygo/framework/pkg/core/cqrs"
	"github.com/fastygo/framework/pkg/web"
)

type Module struct {
	dispatcher       *cqrs.Dispatcher
	navItems         []app.NavItem
	htmlCache        *cache.Cache[[]byte]
	defaultLocale    string
	availableLocales []string
}

func New(dispatcher *cqrs.Dispatcher, defaultLocale string, availableLocales []string) *Module {
	docsNavigation := make([]views.DocsListItem, 0, len(appdocs.Registry()))
	for _, page := range appdocs.Registry() {
		docsNavigation = append(docsNavigation, views.DocsListItem{
			Slug:  page.Slug,
			Title: page.Title,
		})
	}

	navItems := make([]app.NavItem, len(docsNavigation))
	for i, page := range docsNavigation {
		navItems[i] = app.NavItem{
			Label: page.Title,
			Path:  "/" + page.Slug,
			Icon:  "book-open",
			Order: i,
		}
	}

	defaultLocale = normalizeLocale(defaultLocale)
	if defaultLocale == "" {
		defaultLocale = "en"
	}
	availableLocales = normalizeLocales(append([]string{defaultLocale}, availableLocales...))

	return &Module{
		dispatcher:       dispatcher,
		navItems:         navItems,
		htmlCache:        cache.New[[]byte](10 * time.Minute),
		defaultLocale:    defaultLocale,
		availableLocales: availableLocales,
	}
}

func (m *Module) ID() string {
	return "docs"
}

func (m *Module) NavItems() []app.NavItem {
	return append([]app.NavItem{}, m.navItems...)
}

func (m *Module) SetNavItems(items []app.NavItem) {
	m.navItems = append([]app.NavItem{}, items...)
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/", m.handleDocsRoot)
}

func (m *Module) handleDocsRoot(w http.ResponseWriter, r *http.Request) {
	locale := resolveLocale(r, m.defaultLocale, m.availableLocales)

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		m.renderDocsIndex(w, r, locale)
		return
	}
	if strings.Contains(path, "/") {
		http.NotFound(w, r)
		return
	}

	m.renderDocsPage(w, r, path, locale)
}

func (m *Module) renderDocsIndex(w http.ResponseWriter, r *http.Request, locale string) {
	result, err := cqrs.DispatchQuery[appdocs.DocsListQuery, appdocs.DocsListQueryResult](
		r.Context(),
		m.dispatcher,
		appdocs.DocsListQuery{Locale: locale},
	)
	if err != nil {
		web.HandleError(w, err)
		return
	}

	localized, err := m.loadLocalized(locale)
	if err != nil {
		web.HandleError(w, err)
		return
	}

	pages := make([]views.DocsListItem, 0, len(result.Pages))
	for _, page := range result.Pages {
		pages = append(pages, views.DocsListItem{
			Slug:  page.Slug,
			Title: localizePageTitle(page.Slug, page.Title, localized.Common.Pages),
		})
	}

	layout := views.DocsLayoutData{
		Title:     localized.Common.IndexTitle,
		BrandName: localized.Common.BrandName,
		Active:    "/",
		NavItems:  m.localizedNavItems(localized),
		ThemeToggle: web.ThemeToggleData{
			Label:              localized.Common.Theme.Label,
			SwitchToDarkLabel:  localized.Common.Theme.SwitchToDarkLabel,
			SwitchToLightLabel: localized.Common.Theme.SwitchToLightLabel,
		},
		LanguageToggle: localizedLanguageToggle(locale, localized, m.defaultLocale, m.availableLocales),
	}

	if err = web.CachedRender(
		r.Context(),
		w,
		r,
		m.htmlCache,
		"docs:index:"+locale,
		views.DocsLayout(layout, templ.NopComponent, views.DocsIndex(pagesToViewModel(
			pages,
			localized.Common.IndexTitle,
			localized.Common.IndexDescription,
		))),
	); err != nil {
		web.HandleError(w, err)
	}
}

func (m *Module) renderDocsPage(w http.ResponseWriter, r *http.Request, slug string, locale string) {
	result, err := cqrs.DispatchQuery[appdocs.DocsPageQuery, appdocs.DocsPageResult](
		r.Context(),
		m.dispatcher,
		appdocs.DocsPageQuery{
			Slug:   slug,
			Locale: locale,
		},
	)
	if err != nil {
		web.HandleError(w, err)
		return
	}

	localized, err := m.loadLocalized(locale)
	if err != nil {
		web.HandleError(w, err)
		return
	}

	pageTitle := localizePageTitle(result.Slug, result.Title, localized.Common.Pages)

	layout := views.DocsLayoutData{
		Title:     pageTitle,
		BrandName: localized.Common.BrandName,
		Active:    "/" + result.Slug,
		NavItems:  m.localizedNavItems(localized),
		ThemeToggle: web.ThemeToggleData{
			Label:              localized.Common.Theme.Label,
			SwitchToDarkLabel:  localized.Common.Theme.SwitchToDarkLabel,
			SwitchToLightLabel: localized.Common.Theme.SwitchToLightLabel,
		},
		LanguageToggle: localizedLanguageToggle(locale, localized, m.defaultLocale, m.availableLocales),
	}

	pageData := views.DocsPageData{
		Title:       pageTitle,
		HTMLContent: result.HTML,
	}

	if err = web.CachedRender(
		r.Context(),
		w,
		r,
		m.htmlCache,
		"docs:"+locale+":"+slug,
		views.DocsLayout(layout, templ.NopComponent, views.DocsPage(pageData)),
	); err != nil {
		web.HandleError(w, err)
	}
}

func (m *Module) loadLocalized(locale string) (docsi18n.Localized, error) {
	localized, err := docsi18n.Load(locale)
	if err != nil {
		return docsi18n.Load(m.defaultLocale)
	}

	return localized, nil
}

func (m *Module) localizedNavItems(localized docsi18n.Localized) []app.NavItem {
	navItems := make([]app.NavItem, len(m.navItems))
	for i, item := range m.navItems {
		label := item.Label
		slug := strings.TrimPrefix(item.Path, "/")
		if localizedTitle, ok := localized.Common.Pages[slug]; ok && localizedTitle != "" {
			label = localizedTitle
		}

		navItems[i] = app.NavItem{
			Label: label,
			Path:  item.Path,
			Icon:  item.Icon,
			Order: item.Order,
		}
	}

	return navItems
}

func localizePageTitle(slug, fallback string, localized map[string]string) string {
	if localizedTitle, ok := localized[slug]; ok && localizedTitle != "" {
		return localizedTitle
	}
	return fallback
}

func localizedLanguageToggle(currentLocale string, localized docsi18n.Localized, defaultLocale string, configuredLocales []string) partials.LanguageToggleData {
	available := make([]string, 0, len(configuredLocales))
	for _, locale := range configuredLocales {
		if containsLocale(localized.Common.Language.Available, locale) {
			available = append(available, locale)
		}
	}
	if len(available) == 0 {
		available = append(available, defaultLocale)
	}

	nextLocale := localized.Common.Language.NextLocale
	if !containsLocale(available, nextLocale) {
		nextLocale = ""
		for _, locale := range available {
			if locale != currentLocale {
				nextLocale = locale
				break
			}
		}
	}
	if nextLocale == currentLocale {
		nextLocale = ""
	}

	nextLabel := localized.Common.Language.NextLabel
	if nextLabel == "" && nextLocale != "" {
		nextLabel = localized.Common.Language.LocaleLabels[nextLocale]
	}

	currentLabel := localized.Common.Language.CurrentLabel
	if currentLabel == "" {
		currentLabel = localized.Common.LocaleName[currentLocale]
	}
	if currentLabel == "" {
		currentLabel = strings.ToUpper(currentLocale)
	}

	return partials.LanguageToggleData{
		Label:            localized.Common.Language.Label,
		CurrentLocale:    currentLocale,
		CurrentLabel:     currentLabel,
		NextLocale:       nextLocale,
		NextLabel:        nextLabel,
		DefaultLocale:    defaultLocale,
		AvailableLocales: available,
	}
}

func resolveLocale(r *http.Request, defaultLocale string, availableLocales []string) string {
	locale := normalizeLocale(r.URL.Query().Get("lang"))
	if locale == "" {
		return defaultLocale
	}
	if len(locale) > 2 {
		locale = locale[:2]
	}
	if containsLocale(availableLocales, locale) {
		return locale
	}
	return defaultLocale
}

func normalizeLocale(locale string) string {
	return strings.ToLower(strings.TrimSpace(locale))
}

func normalizeLocales(locales []string) []string {
	normalized := make([]string, 0, len(locales))
	seen := map[string]struct{}{}
	for _, locale := range locales {
		normalizedLocale := normalizeLocale(locale)
		if normalizedLocale == "" {
			continue
		}
		if _, ok := seen[normalizedLocale]; ok {
			continue
		}
		seen[normalizedLocale] = struct{}{}
		normalized = append(normalized, normalizedLocale)
	}
	if len(normalized) == 0 {
		normalized = []string{"en"}
	}
	return normalized
}

func containsLocale(locales []string, locale string) bool {
	for _, available := range locales {
		if available == locale {
			return true
		}
	}
	return false
}

func pagesToViewModel(pages []views.DocsListItem, title string, description string) views.DocsIndexData {
	return views.DocsIndexData{
		Title:       title,
		Description: description,
		Pages:       pages,
	}
}
