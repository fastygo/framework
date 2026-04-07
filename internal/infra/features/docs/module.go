package docs

import (
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	appdocs "github.com/fastygo/framework/internal/application/docs"
	"github.com/fastygo/framework/internal/site/docs/web/views"
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/cache"
	"github.com/fastygo/framework/pkg/core/cqrs"
	"github.com/fastygo/framework/pkg/web"
)

type Module struct {
	dispatcher     *cqrs.Dispatcher
	navItems       []app.NavItem
	htmlCache      *cache.Cache[[]byte]
}

func New(dispatcher *cqrs.Dispatcher) *Module {
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

	return &Module{
		dispatcher:     dispatcher,
		navItems:       navItems,
		htmlCache:      cache.New[[]byte](10 * time.Minute),
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
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		m.renderDocsIndex(w, r)
		return
	}
	if strings.Contains(path, "/") {
		http.NotFound(w, r)
		return
	}

	m.renderDocsPage(w, r, path)
}

func (m *Module) renderDocsIndex(w http.ResponseWriter, r *http.Request) {
	result, err := cqrs.DispatchQuery[appdocs.DocsListQuery, appdocs.DocsListQueryResult](r.Context(), m.dispatcher, appdocs.DocsListQuery{})
	if err != nil {
		web.HandleError(w, err)
		return
	}

	pages := make([]views.DocsListItem, 0, len(result.Pages))
	for _, page := range result.Pages {
		pages = append(pages, views.DocsListItem{
			Slug:  page.Slug,
			Title: page.Title,
		})
	}

	layout := views.DocsLayoutData{
		Title:     "Framework Docs",
		BrandName: "Framework Docs",
		Active:    "/",
		NavItems:  m.navItems,
		ThemeToggle: web.ThemeToggleData{
			Label:             "Theme",
			SwitchToDarkLabel: "Switch to dark theme",
			SwitchToLightLabel: "Switch to light theme",
		},
	}

	if err = web.CachedRender(
		r.Context(),
		w,
		r,
		m.htmlCache,
		"docs:index",
		views.DocsLayout(layout, templ.NopComponent, views.DocsIndex(pagesToViewModel(pages))),
	); err != nil {
		web.HandleError(w, err)
	}
}

func (m *Module) renderDocsPage(w http.ResponseWriter, r *http.Request, slug string) {
	result, err := cqrs.DispatchQuery[appdocs.DocsPageQuery, appdocs.DocsPageResult](
		r.Context(),
		m.dispatcher,
		appdocs.DocsPageQuery{Slug: slug},
	)
	if err != nil {
		web.HandleError(w, err)
		return
	}

	layout := views.DocsLayoutData{
		Title:     result.Title,
		BrandName: "Framework Docs",
		Active:    "/" + result.Slug,
		NavItems:  m.navItems,
		ThemeToggle: web.ThemeToggleData{
			Label:             "Theme",
			SwitchToDarkLabel: "Switch to dark theme",
			SwitchToLightLabel: "Switch to light theme",
		},
	}

	pageData := views.DocsPageData{
		Title:       result.Title,
		HTMLContent: result.HTML,
	}

	if err = web.CachedRender(
		r.Context(),
		w,
		r,
		m.htmlCache,
		"docs:"+slug,
		views.DocsLayout(layout, templ.NopComponent, views.DocsPage(pageData)),
	); err != nil {
		web.HandleError(w, err)
	}
}

func pagesToViewModel(pages []views.DocsListItem) views.DocsIndexData {
	return views.DocsIndexData{Pages: pages}
}
