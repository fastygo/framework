package home

import (
	"net/http"
	"time"

	"github.com/a-h/templ"

	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/cache"
	"github.com/fastygo/framework/pkg/web"
	"github.com/fastygo/framework/pkg/web/locale"
	"github.com/fastygo/framework/pkg/web/view"

	"github.com/fastygo/framework/examples/storefront/internal/site/i18n"
	"github.com/fastygo/framework/examples/storefront/internal/site/views"
)

type Feature struct {
	htmlCache *cache.Cache[[]byte]
	navItems  []app.NavItem
	merged    []app.NavItem
}

func New() *Feature {
	return &Feature{
		htmlCache: cache.New[[]byte](5 * time.Minute),
	}
}

func (f *Feature) ID() string { return "storefront" }

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
	bundle, err := i18n.Load(loc)
	if err != nil {
		web.HandleError(w, err)
		return
	}

	f.navItems = navItems(bundle.Storefront.Nav)
	headerNav := f.merged
	if len(headerNav) == 0 {
		headerNav = f.navItems
	}

	language := view.BuildLanguageToggleFromContext(r.Context(),
		view.WithLabel(bundle.Storefront.Language.Label),
		view.WithCurrentLabel(bundle.Storefront.Language.CurrentLabel),
		view.WithNextLocale(bundle.Storefront.Language.NextLocale),
		view.WithNextLabel(bundle.Storefront.Language.NextLabel),
		view.WithLocaleLabels(bundle.Storefront.Language.LocaleLabels),
	)

	layout := views.LayoutData{
		Title:          bundle.Storefront.Title,
		Locale:         loc,
		Active:         "/",
		BrandName:      bundle.Storefront.BrandName,
		NavItems:       f.navItems,
		HeaderNavItems: headerNav,
		ThemeToggle: view.ThemeToggleData{
			Label:              bundle.Storefront.Theme.Label,
			SwitchToDarkLabel:  bundle.Storefront.Theme.SwitchToDarkLabel,
			SwitchToLightLabel: bundle.Storefront.Theme.SwitchToLightLabel,
		},
		LanguageToggle: language,
		HeaderActions: views.HeaderActionsData{
			SearchLabel:  bundle.Storefront.Header.SearchLabel,
			AccountLabel: bundle.Storefront.Header.AccountLabel,
			CartLabel:    bundle.Storefront.Header.CartLabel,
			CartCount:    bundle.Storefront.Header.CartCount,
		},
	}

	page := pageData(bundle.Storefront)

	if err := web.CachedRender(
		r.Context(),
		w,
		r,
		f.htmlCache,
		"storefront:"+loc,
		views.Layout(layout, templ.NopComponent, views.StorefrontPage(page)),
	); err != nil {
		web.HandleError(w, err)
	}
}

func navItems(items []i18n.ActionFixture) []app.NavItem {
	out := make([]app.NavItem, 0, len(items))
	for i, item := range items {
		out = append(out, app.NavItem{
			Label: item.Label,
			Path:  item.Href,
			Order: i,
		})
	}
	return out
}

func pageData(f i18n.StorefrontFixture) views.Page {
	return views.Page{
		Hero:        heroData(f.Hero),
		Categories:  categories(f.Categories),
		Collections: collections(f.Collections),
		Benefits:    benefits(f.Benefits),
		Product:     product(f.Product),
		Rail:        rail(f.Rail),
		Footer: views.Footer{
			BrandName: f.BrandName,
			Copyright: f.Footer.Copyright,
			Groups:    footerGroups(f.Footer.Groups),
		},
	}
}

func heroData(hero i18n.HeroFixture) views.Hero {
	return views.Hero{
		Kicker:       hero.Kicker,
		Title:        hero.Title,
		Description:  hero.Description,
		Primary:      action(hero.Primary),
		SlideCurrent: hero.SlideCurrent,
		SlideTotal:   hero.SlideTotal,
		Previous:     action(hero.Previous),
		Next:         action(hero.Next),
		Image:        image(hero.Image),
	}
}

func categories(items []i18n.Category) []views.Category {
	out := make([]views.Category, 0, len(items))
	for _, item := range items {
		out = append(out, views.Category{Title: item.Title, Count: item.Count, Href: item.Href, Image: image(item.Image)})
	}
	return out
}

func collections(items []i18n.Collection) []views.Collection {
	out := make([]views.Collection, 0, len(items))
	for _, item := range items {
		out = append(out, views.Collection{
			Label:       item.Label,
			Title:       item.Title,
			Description: item.Description,
			Href:        item.Href,
			Image:       image(item.Image),
		})
	}
	return out
}

func benefits(items []i18n.Benefit) []views.Benefit {
	out := make([]views.Benefit, 0, len(items))
	for _, item := range items {
		out = append(out, views.Benefit{Icon: item.Icon, Title: item.Title, Description: item.Description})
	}
	return out
}

func product(item i18n.Product) views.Product {
	return views.Product{
		Badge:          item.Badge,
		Name:           item.Name,
		Price:          item.Price,
		OldPrice:       item.OldPrice,
		Description:    item.Description,
		Gallery:        images(item.Gallery),
		MainImage:      image(item.MainImage),
		ColorsLabel:    item.ColorsLabel,
		Colors:         swatches(item.Colors),
		SelectedColor:  item.SelectedColor,
		Quantity:       quantity(item.Quantity),
		AddAction:      action(item.AddAction),
		FavoriteAction: action(item.FavoriteAction),
		Details:        details(item.Details),
	}
}

func rail(item i18n.Rail) views.Rail {
	return views.Rail{
		Cart:        cart(item.Cart),
		Filter:      filter(item.Filter),
		Inspiration: inspiration(item.Inspiration),
		Newsletter: views.Newsletter{
			Title:       item.Newsletter.Title,
			Description: item.Newsletter.Description,
			Placeholder: item.Newsletter.Placeholder,
			SubmitLabel: item.Newsletter.SubmitLabel,
		},
		Socials: actions(item.Socials),
	}
}

func cart(item i18n.Cart) views.Cart {
	return views.Cart{
		Title:          item.Title,
		CountLabel:     item.CountLabel,
		Items:          cartItems(item.Items),
		TotalLabel:     item.TotalLabel,
		Total:          item.Total,
		CheckoutAction: action(item.CheckoutAction),
		ViewCartAction: action(item.ViewCartAction),
	}
}

func cartItems(items []i18n.CartItem) []views.CartItem {
	out := make([]views.CartItem, 0, len(items))
	for _, item := range items {
		out = append(out, views.CartItem{Title: item.Title, Subtitle: item.Subtitle, Price: item.Price, Image: image(item.Image)})
	}
	return out
}

func filter(item i18n.Filter) views.Filter {
	return views.Filter{
		Title:         item.Title,
		ResetLabel:    item.ResetLabel,
		CategoryLabel: item.CategoryLabel,
		CategoryValue: item.CategoryValue,
		MaterialLabel: item.MaterialLabel,
		Materials:     swatches(item.Materials),
		Price: views.PriceRange{
			Label:    item.Price.Label,
			MinLabel: item.Price.MinLabel,
			MaxLabel: item.Price.MaxLabel,
			Value:    item.Price.Value,
		},
		ColorLabel:    item.ColorLabel,
		Colors:        swatches(item.Colors),
		ShowAllAction: action(item.ShowAllAction),
	}
}

func inspiration(item i18n.Inspiration) views.Inspiration {
	return views.Inspiration{
		Title:       item.Title,
		Description: item.Description,
		Action:      action(item.Action),
		Image:       image(item.Image),
	}
}

func footerGroups(groups []i18n.FooterGroup) []views.FooterGroup {
	out := make([]views.FooterGroup, 0, len(groups))
	for _, group := range groups {
		out = append(out, views.FooterGroup{Title: group.Title, Links: actions(group.Links)})
	}
	return out
}

func images(items []i18n.ImageFixture) []views.Image {
	out := make([]views.Image, 0, len(items))
	for _, item := range items {
		out = append(out, image(item))
	}
	return out
}

func image(item i18n.ImageFixture) views.Image {
	return views.Image{Alt: item.Alt}
}

func swatches(items []i18n.SwatchFixture) []views.Swatch {
	out := make([]views.Swatch, 0, len(items))
	for _, item := range items {
		out = append(out, views.Swatch{
			Label:    item.Label,
			Value:    item.Value,
			Class:    item.Class,
			Selected: item.Selected,
		})
	}
	return out
}

func quantity(item i18n.Quantity) views.Quantity {
	return views.Quantity{
		Label:      item.Label,
		Value:      item.Value,
		MinusLabel: item.MinusLabel,
		PlusLabel:  item.PlusLabel,
	}
}

func details(items []i18n.Detail) []views.Detail {
	out := make([]views.Detail, 0, len(items))
	for _, item := range items {
		out = append(out, views.Detail{Label: item.Label, Value: item.Value})
	}
	return out
}

func actions(items []i18n.ActionFixture) []views.Action {
	out := make([]views.Action, 0, len(items))
	for _, item := range items {
		out = append(out, action(item))
	}
	return out
}

func action(item i18n.ActionFixture) views.Action {
	return views.Action{
		Label:     item.Label,
		Href:      item.Href,
		AriaLabel: item.AriaLabel,
	}
}
