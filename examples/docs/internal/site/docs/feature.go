package docs

import (
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"

	docsblocks "github.com/fastygo/blocks/docs"
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/cache"
	content "github.com/fastygo/framework/pkg/content-markdown"
	"github.com/fastygo/framework/pkg/web"
	"github.com/fastygo/framework/pkg/web/locale"
	"github.com/fastygo/framework/pkg/web/view"

	"github.com/fastygo/framework/examples/docs/internal/site/i18n"
	"github.com/fastygo/framework/examples/docs/internal/site/views"
)

// Pages declares the documentation pages exposed by this site.
//
// To add a new page:
//  1. Drop the markdown file at content/i18n/<locale>/<slug>.md.
//  2. Add a {Slug, Title} entry below.
//  3. Optionally translate the title via web/i18n/<locale>/common.json.
var Pages = []content.PageMeta{
	{Slug: "quickstart", Title: "Quickstart"},
	{Slug: "developer-guide", Title: "Developer Guide"},
	{Slug: "api-reference", Title: "API Reference"},
}

// Feature wires the docs site routes.
type Feature struct {
	library   *content.Library
	htmlCache *cache.Cache[[]byte]
	navItems  []app.NavItem
}

// New constructs a docs feature backed by an in-memory rendered library.
func New(library *content.Library) *Feature {
	navItems := make([]app.NavItem, 0, len(library.Pages()))
	for i, page := range Pages {
		navItems = append(navItems, app.NavItem{
			Label: page.Title,
			Path:  "/" + page.Slug,
			Icon:  "book-open",
			Order: i,
		})
	}
	return &Feature{
		library:   library,
		htmlCache: cache.New[[]byte](10 * time.Minute),
		navItems:  navItems,
	}
}

func (f *Feature) ID() string                { return "docs" }
func (f *Feature) NavItems() []app.NavItem   { return cloneNav(f.navItems) }
func (f *Feature) SetNavItems([]app.NavItem) {}

func (f *Feature) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/", f.handle)
}

func (f *Feature) handle(w http.ResponseWriter, r *http.Request) {
	loc := locale.From(r.Context())

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		f.renderIndex(w, r, loc)
		return
	}
	if strings.Contains(path, "/") {
		http.NotFound(w, r)
		return
	}
	f.renderPage(w, r, loc, path)
}

func (f *Feature) renderIndex(w http.ResponseWriter, r *http.Request, loc string) {
	bundle, err := i18n.Load(loc)
	if err != nil {
		web.HandleError(w, err)
		return
	}

	pages := make([]docsblocks.DocListItem, 0, len(Pages))
	for _, page := range Pages {
		title := page.Title
		if t, ok := bundle.Common.Pages[page.Slug]; ok && t != "" {
			title = t
		}
		pages = append(pages, docsblocks.DocListItem{Slug: page.Slug, Title: title})
	}

	layout := f.layoutFor(r, bundle, loc, "/")
	layout.Title = bundle.Common.IndexTitle

	if err := web.CachedRender(
		r.Context(),
		w,
		r,
		f.htmlCache,
		"docs:index:"+loc,
		views.DocsLayout(layout, templ.NopComponent, docsblocks.DocsIndex(docsblocks.DocsIndexProps{
			Title:       bundle.Common.IndexTitle,
			Description: bundle.Common.IndexDescription,
			Pages:       pages,
		})),
	); err != nil {
		web.HandleError(w, err)
	}
}

func (f *Feature) renderPage(w http.ResponseWriter, r *http.Request, loc, slug string) {
	rendered, ok := f.library.Page(loc, slug)
	if !ok {
		http.NotFound(w, r)
		return
	}

	bundle, err := i18n.Load(loc)
	if err != nil {
		web.HandleError(w, err)
		return
	}

	pageTitle := rendered.Title
	if t, ok := bundle.Common.Pages[slug]; ok && t != "" {
		pageTitle = t
	}

	layout := f.layoutFor(r, bundle, loc, "/"+slug)
	layout.Title = pageTitle

	if err := web.CachedRender(
		r.Context(),
		w,
		r,
		f.htmlCache,
		"docs:"+loc+":"+slug,
		views.DocsLayout(layout, templ.NopComponent, docsblocks.DocsArticle(docsblocks.DocsArticleProps{
			HTMLContent: rendered.HTML,
		})),
	); err != nil {
		web.HandleError(w, err)
	}
}

func (f *Feature) layoutFor(r *http.Request, bundle i18n.Bundle, loc, active string) views.DocsLayoutData {
	available := locale.Available(r.Context())
	if len(available) == 0 {
		available = i18n.Locales
	}

	headerNav := make([]app.NavItem, 0, len(bundle.Common.HeaderNavItems))
	for _, link := range bundle.Common.HeaderNavItems {
		headerNav = append(headerNav, app.NavItem{Label: link.Label, Path: link.Path})
	}

	nav := make([]app.NavItem, 0, len(f.navItems))
	for _, item := range f.navItems {
		label := item.Label
		slug := strings.TrimPrefix(item.Path, "/")
		if t, ok := bundle.Common.Pages[slug]; ok && t != "" {
			label = t
		}
		nav = append(nav, app.NavItem{Label: label, Path: item.Path, Icon: item.Icon, Order: item.Order})
	}

	currentLabel := bundle.Common.Language.CurrentLabel
	if currentLabel == "" {
		currentLabel = bundle.Common.LocaleName[loc]
	}

	language := view.BuildLanguageToggleFromContext(r.Context(),
		view.WithAvailable(available),
		view.WithLabel(bundle.Common.Language.Label),
		view.WithCurrentLabel(currentLabel),
		view.WithNextLocale(bundle.Common.Language.NextLocale),
		view.WithNextLabel(bundle.Common.Language.NextLabel),
		view.WithLocaleLabels(bundle.Common.Language.LocaleLabels),
	)

	return views.DocsLayoutData{
		BrandName:      bundle.Common.BrandName,
		Active:         active,
		NavItems:       nav,
		HeaderNavItems: headerNav,
		ThemeToggle: view.ThemeToggleData{
			Label:              bundle.Common.Theme.Label,
			SwitchToDarkLabel:  bundle.Common.Theme.SwitchToDarkLabel,
			SwitchToLightLabel: bundle.Common.Theme.SwitchToLightLabel,
		},
		LanguageToggle: language,
	}
}

func cloneNav(items []app.NavItem) []app.NavItem {
	out := make([]app.NavItem, len(items))
	copy(out, items)
	return out
}
