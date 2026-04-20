package welcome

import (
	"context"
	"net/http"
	"time"

	"github.com/a-h/templ"

	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/cache"
	"github.com/fastygo/framework/pkg/core/cqrs"
	"github.com/fastygo/framework/pkg/web"
	"github.com/fastygo/framework/pkg/web/locale"
	"github.com/fastygo/framework/pkg/web/view"

	"github.com/fastygo/framework/examples/web/internal/site/i18n"
	"github.com/fastygo/framework/examples/web/internal/site/views"
)

// QueryHandler is the read-side of the welcome page (loads localized strings).
type QueryHandler struct{}

// NewQueryHandler constructs a stateless query handler.
func NewQueryHandler() QueryHandler { return QueryHandler{} }

type Query struct {
	Locale string
}

func (q Query) Validate() error { return nil }

type QueryResult struct {
	Bundle i18n.Bundle
}

func (QueryHandler) Handle(_ context.Context, q Query) (QueryResult, error) {
	bundle, err := i18n.Load(q.Locale)
	if err != nil {
		return QueryResult{}, err
	}
	return QueryResult{Bundle: bundle}, nil
}

// Feature is the welcome HTTP feature wired into AppBuilder.
type Feature struct {
	dispatcher    *cqrs.Dispatcher
	htmlCache     *cache.Cache[[]byte]
	navItems      []app.NavItem
	merged        []app.NavItem
}

// New constructs a welcome feature.
func New(dispatcher *cqrs.Dispatcher) *Feature {
	return &Feature{
		dispatcher: dispatcher,
		htmlCache:     cache.New[[]byte](5 * time.Minute),
		navItems: []app.NavItem{
			{Label: "Home", Path: "/", Icon: "home", Order: 0},
			{Label: "Docs", Path: "https://docs.fastygo.ru", Icon: "book-open", Order: 90},
		},
	}
}

func (f *Feature) ID() string { return "welcome" }

func (f *Feature) NavItems() []app.NavItem {
	out := make([]app.NavItem, len(f.navItems))
	copy(out, f.navItems)
	return out
}

func (f *Feature) SetNavItems(items []app.NavItem) {
	f.merged = append(f.merged[:0], items...)
}

func (f *Feature) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/", f.handle)
}

func (f *Feature) handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	loc := locale.From(r.Context())
	result, err := cqrs.DispatchQuery[Query, QueryResult](r.Context(), f.dispatcher, Query{Locale: loc})
	if err != nil {
		web.HandleError(w, err)
		return
	}

	common := result.Bundle.Common
	welcome := result.Bundle.Welcome

	language := view.BuildLanguageToggleFromContext(r.Context(),
		view.WithLabel(common.Language.Label),
		view.WithCurrentLabel(common.Language.CurrentLabel),
		view.WithNextLocale(common.Language.NextLocale),
		view.WithNextLabel(common.Language.NextLabel),
		view.WithLocaleLabels(common.Language.LocaleLabels),
	)

	headerNav := f.merged
	if headerNav == nil {
		headerNav = f.navItems
	}

	layout := views.LayoutData{
		Title:          welcome.Title,
		Locale:         loc,
		Active:         "/",
		BrandName:      common.BrandName,
		NavItems:       f.merged,
		HeaderNavItems: headerNav,
		ThemeToggle: view.ThemeToggleData{
			Label:              common.Theme.Label,
			SwitchToDarkLabel:  common.Theme.SwitchToDarkLabel,
			SwitchToLightLabel: common.Theme.SwitchToLightLabel,
		},
		LanguageToggle: language,
	}

	page := views.WelcomePageData{
		Title:                 welcome.Title,
		Description:           welcome.Description,
		ButtonLabel:           welcome.ButtonLabel,
		Kicker:                welcome.Kicker,
		ModularTitle:          welcome.ModularTitle,
		ModularDescription:    welcome.ModularDescription,
		BootstrapTitle:        welcome.BootstrapTitle,
		BootstrapDescription:  welcome.BootstrapDescription,
		ProductionTitle:       welcome.ProductionTitle,
		ProductionDescription: welcome.ProductionDescription,
		GithubLabel:           welcome.GithubLabel,
		DocsLabel:             welcome.DocsLabel,
	}

	if err := web.CachedRender(
		r.Context(),
		w,
		r,
		f.htmlCache,
		"welcome:"+loc,
		views.Layout(layout, templ.NopComponent, views.WelcomePage(page)),
	); err != nil {
		web.HandleError(w, err)
	}
}
