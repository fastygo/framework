// Package blog implements the blog feature of the example.
package blog

import (
	"net/http"
	"strings"
	"time"

	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/cache"
	content "github.com/fastygo/framework/pkg/content-markdown"
	"github.com/fastygo/framework/pkg/web"
	"github.com/fastygo/framework/pkg/web/locale"
	"github.com/fastygo/framework/pkg/web/view"

	blogi18n "github.com/fastygo/framework/examples/blog/internal/site/i18n"
	"github.com/fastygo/framework/examples/blog/internal/site/views"
)

// Posts declares the blog content registry.
//
// Each entry must match a markdown file at content/i18n/<locale>/<slug>.md.
// The Summary is rendered on the index page; keep it short.
type Post struct {
	Slug    string
	Title   string
	Summary string
}

var Posts = []Post{
	{
		Slug:    "hello-world",
		Title:   "Hello, world",
		Summary: "Why this blog exists and how new posts are added.",
	},
	{
		Slug:    "architecture-notes",
		Title:   "Architecture notes",
		Summary: "How the framework's content library, locale negotiator, and cache plug together.",
	},
	{
		Slug:    "why-ui8kit",
		Title:   "Why UI8Kit",
		Summary: "What you get for free when adopting the UI8Kit shell pattern.",
	},
}

// Feature is the public blog HTTP feature.
type Feature struct {
	library  *content.Library
	cache    *cache.Cache[[]byte]
	posts    []Post
	brand    string
	header   []app.NavItem
	theme    view.ThemeToggleData
	navItems []app.NavItem
}

// Options groups optional behaviour for the blog feature.
type Options struct {
	BrandName      string
	HeaderNavItems []app.NavItem
	Theme          view.ThemeToggleData
}

// New constructs a blog feature backed by an in-memory rendered library.
func New(library *content.Library, opts Options) *Feature {
	if opts.BrandName == "" {
		opts.BrandName = "Acme Blog"
	}
	if opts.Theme.Label == "" {
		opts.Theme = view.ThemeToggleData{
			Label:              "Theme",
			SwitchToDarkLabel:  "Switch to dark mode",
			SwitchToLightLabel: "Switch to light mode",
		}
	}
	if len(opts.HeaderNavItems) == 0 {
		opts.HeaderNavItems = []app.NavItem{
			{Label: "Home", Path: "/"},
			{Label: "Framework", Path: "https://github.com/fastygo/framework"},
		}
	}

	return &Feature{
		library:  library,
		cache:    cache.New[[]byte](10 * time.Minute),
		posts:    append([]Post(nil), Posts...),
		brand:    opts.BrandName,
		header:   opts.HeaderNavItems,
		theme:    opts.Theme,
		navItems: deriveNav(Posts),
	}
}

func (f *Feature) ID() string              { return "blog" }
func (f *Feature) NavItems() []app.NavItem { return cloneNav(f.navItems) }

func (f *Feature) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", f.handleIndex)
	mux.HandleFunc("GET /posts/{slug}", f.handlePost)
}

func (f *Feature) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc := locale.From(r.Context())
	bundle, err := blogi18n.Load(loc)
	if err != nil {
		bundle = blogi18n.Bundle{}
	}

	brandName := f.brand
	if bundle.Common.BrandName != "" {
		brandName = bundle.Common.BrandName
	}

	theme := mergeTheme(f.theme, bundle.Common.Theme)
	headerNav := f.header
	if len(bundle.Common.HeaderNav) > 0 {
		headerNav = toAppNavItems(bundle.Common.HeaderNav)
	}

	items := make([]views.PostListItem, 0, len(f.posts))
	for _, post := range f.posts {
		items = append(items, views.PostListItem{
			Slug:    post.Slug,
			Title:   post.Title,
			Summary: post.Summary,
		})
	}

	layout := views.LayoutData{
		Title:          localizedTitle(bundle.Common.IndexTitle, "Latest posts", brandName),
		Lang:           loc,
		Active:         "/",
		BrandName:      brandName,
		NavItems:       f.navItems,
		HeaderNavItems: headerNav,
		ThemeToggle:    theme,
		LanguageToggle: f.buildLanguageToggle(r, bundle),
	}

	indexTitle := localizedTitle(bundle.Common.IndexTitle, "Latest posts", brandName)
	indexDescription := bundle.Common.IndexDescription
	if indexDescription == "" {
		indexDescription = "Long-form notes on building with fastygo + ui8kit."
	}

	if err := web.CachedRender(
		r.Context(),
		w,
		r,
		f.cache,
		"blog:index:"+loc,
		views.BlogLayout(layout, views.PostList(views.PostListData{
			Title:       indexTitle,
			Description: indexDescription,
			Posts:       items,
		})),
	); err != nil {
		web.HandleError(w, err)
	}
}

func (f *Feature) handlePost(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(r.PathValue("slug"))
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	loc := locale.From(r.Context())
	bundle, err := blogi18n.Load(loc)
	if err != nil {
		bundle = blogi18n.Bundle{}
	}

	rendered, ok := f.library.Page(loc, slug)
	if !ok {
		http.NotFound(w, r)
		return
	}

	post := f.findPost(slug)
	pageTitle := rendered.Title
	if post != nil {
		pageTitle = post.Title
	}

	brandName := f.brand
	if bundle.Common.BrandName != "" {
		brandName = bundle.Common.BrandName
	}

	theme := mergeTheme(f.theme, bundle.Common.Theme)
	headerNav := f.header
	if len(bundle.Common.HeaderNav) > 0 {
		headerNav = toAppNavItems(bundle.Common.HeaderNav)
	}

	layout := views.LayoutData{
		Title:          pageTitle + " — " + brandName,
		Lang:           loc,
		Active:         "/posts/" + slug,
		BrandName:      brandName,
		NavItems:       f.navItems,
		HeaderNavItems: headerNav,
		ThemeToggle:    theme,
		LanguageToggle: f.buildLanguageToggle(r, bundle),
	}

	if err := web.CachedRender(
		r.Context(),
		w,
		r,
		f.cache,
		"blog:post:"+slug+":"+loc,
		views.BlogLayout(layout, views.PostPage(views.PostPageData{
			Title:       pageTitle,
			HTMLContent: rendered.HTML,
		})),
	); err != nil {
		web.HandleError(w, err)
	}
}

func (f *Feature) findPost(slug string) *Post {
	for i := range f.posts {
		if f.posts[i].Slug == slug {
			return &f.posts[i]
		}
	}
	return nil
}

func deriveNav(posts []Post) []app.NavItem {
	items := make([]app.NavItem, 0, len(posts)+1)
	items = append(items, app.NavItem{Label: "All posts", Path: "/", Icon: "book-open", Order: 0})
	for i, post := range posts {
		items = append(items, app.NavItem{
			Label: post.Title,
			Path:  "/posts/" + post.Slug,
			Icon:  "file",
			Order: i + 1,
		})
	}
	return items
}

func cloneNav(items []app.NavItem) []app.NavItem {
	out := make([]app.NavItem, len(items))
	copy(out, items)
	return out
}

func (f *Feature) buildLanguageToggle(r *http.Request, bundle blogi18n.Bundle) view.LanguageToggleData {
	available := locale.Available(r.Context())
	if len(available) == 0 {
		available = blogi18n.Locales
	}

	currentLabel := strings.TrimSpace(bundle.Common.Language.CurrentLabel)
	if currentLabel == "" {
		if label, ok := bundle.Common.Language.LocaleLabels[locale.From(r.Context())]; ok && strings.TrimSpace(label) != "" {
			currentLabel = label
		}
	}

	return view.BuildLanguageToggleFromContext(r.Context(),
		view.WithAvailable(available),
		view.WithLabel(bundle.Common.Language.Label),
		view.WithCurrentLabel(currentLabel),
		view.WithNextLocale(bundle.Common.Language.NextLocale),
		view.WithNextLabel(bundle.Common.Language.NextLabel),
		view.WithLocaleLabels(bundle.Common.Language.LocaleLabels),
	)
}

func localizedTitle(translated, fallback, brand string) string {
	title := strings.TrimSpace(translated)
	if title == "" {
		title = fallback
	}
	return title + " — " + brand
}

func mergeTheme(fallback view.ThemeToggleData, themed blogi18n.ThemeFixture) view.ThemeToggleData {
	out := fallback
	if themed.Label != "" {
		out.Label = themed.Label
	}
	if themed.SwitchToDarkLabel != "" {
		out.SwitchToDarkLabel = themed.SwitchToDarkLabel
	}
	if themed.SwitchToLightLabel != "" {
		out.SwitchToLightLabel = themed.SwitchToLightLabel
	}
	return out
}

func toAppNavItems(items []blogi18n.NavLinkFixture) []app.NavItem {
	out := make([]app.NavItem, len(items))
	for i, item := range items {
		out[i] = app.NavItem{
			Label: item.Label,
			Path:  item.Path,
		}
	}
	return out
}

// PageMetas converts the blog post registry into the format expected by
// content.LibraryOptions.
func PageMetas() []content.PageMeta {
	out := make([]content.PageMeta, len(Posts))
	for i, post := range Posts {
		out[i] = content.PageMeta{Slug: post.Slug, Title: post.Title}
	}
	return out
}
