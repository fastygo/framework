package pwa

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/a-h/templ"

	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/cache"
	"github.com/fastygo/framework/pkg/web"
	"github.com/fastygo/framework/pkg/web/locale"
	"github.com/fastygo/framework/pkg/web/view"

	pwasti18n "github.com/fastygo/framework/examples/pwa/internal/site/i18n"
	"github.com/fastygo/framework/examples/pwa/internal/site/views"
)

const (
	htmlCacheTTL        = 10 * time.Minute
	htmlCacheMaxEntries = 32
)

type Feature struct {
	cache    *cache.Cache[[]byte]
	navItems []app.NavItem
	merged   []app.NavItem
	manifest []byte
	worker   []byte
}

func New(assetDir string) (*Feature, error) {
	manifest, err := os.ReadFile(filepath.Join(assetDir, "manifest.webmanifest"))
	if err != nil {
		return nil, err
	}
	worker, err := os.ReadFile(filepath.Join(assetDir, "sw.js"))
	if err != nil {
		return nil, err
	}

	return &Feature{
		cache: cache.NewWithOptions[[]byte](htmlCacheTTL, cache.Options{
			MaxEntries: htmlCacheMaxEntries,
		}),
		navItems: []app.NavItem{
			{Label: "Dashboard", Path: "/", Icon: "home", Order: 0},
			{Label: "Onboarding", Path: "/onboarding", Icon: "sparkles", Order: 10},
			{Label: "Premium", Path: "/paywall", Icon: "crown", Order: 20},
			{Label: "Subscription", Path: "/subscription", Icon: "settings", Order: 30},
		},
		manifest: manifest,
		worker:   worker,
	}, nil
}

func (f *Feature) ID() string { return "pwa" }

func (f *Feature) NavItems() []app.NavItem {
	return cloneNav(f.navItems)
}

func (f *Feature) SetNavItems(items []app.NavItem) {
	f.merged = cloneNav(items)
}

func (f *Feature) BackgroundTasks() []app.BackgroundTask {
	return []app.BackgroundTask{
		app.CleanupTask("pwa-html-cache-cleanup", time.Minute, f.cache),
	}
}

func (f *Feature) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", f.handleHome)
	mux.HandleFunc("GET /onboarding", f.handleOnboarding)
	mux.HandleFunc("GET /paywall", f.handlePaywall)
	mux.HandleFunc("GET /pricing", f.handlePricing)
	mux.HandleFunc("GET /payment", f.handlePayment)
	mux.HandleFunc("GET /payment/processing", f.handleProcessing)
	mux.HandleFunc("GET /payment/success", f.handleSuccess)
	mux.HandleFunc("GET /subscription", f.handleSubscription)
	mux.HandleFunc("GET /subscription/cancel", f.handleCancel)
	mux.HandleFunc("GET /labs/pomodoro", f.handlePomodoro)
	mux.HandleFunc("GET /offline", f.handleOffline)
	mux.HandleFunc("GET /manifest.webmanifest", f.handleManifest)
	mux.HandleFunc("GET /sw.js", f.handleServiceWorker)
}

func (f *Feature) handleHome(w http.ResponseWriter, r *http.Request) {
	f.render(w, r, "/", "home", func(bundle pwasti18n.Bundle) (string, templ.Component) {
		home := bundle.App.Home
		return home.Title, views.HomePage(views.HomePageData{
			Header:       views.PageHeaderData{Kicker: home.Kicker, Title: home.Title, Subtitle: home.Subtitle},
			PrimaryCTA:   home.PrimaryCTA,
			SecondaryCTA: home.SecondaryCTA,
			Tasks:        taskItems(home.Tasks),
			Stats:        statItems(home.Stats),
			Benefits:     textItems(home.Benefits),
			TaskTools:    views.TaskTools(home.TaskTools),
		})
	})
}

func (f *Feature) handleOnboarding(w http.ResponseWriter, r *http.Request) {
	f.render(w, r, "/onboarding", "onboarding", func(bundle pwasti18n.Bundle) (string, templ.Component) {
		page := bundle.App.Onboarding
		return page.Title, views.OnboardingPage(views.OnboardingPageData{
			Header:      views.PageHeaderData{Title: page.Title, Subtitle: page.Subtitle},
			ContinueCTA: page.ContinueCTA,
			SkipCTA:     page.SkipCTA,
			Steps:       stepItems(page.Steps),
		})
	})
}

func (f *Feature) handlePaywall(w http.ResponseWriter, r *http.Request) {
	f.render(w, r, "/paywall", "paywall", func(bundle pwasti18n.Bundle) (string, templ.Component) {
		page := bundle.App.Paywall
		return page.Title, views.PaywallPage(views.PaywallPageData{
			Header:     views.PageHeaderData{Title: page.Title, Subtitle: page.Subtitle},
			Badge:      page.Badge,
			PrimaryCTA: page.PrimaryCTA,
			RestoreCTA: page.RestoreCTA,
			Benefits:   textItems(page.Benefits),
		})
	})
}

func (f *Feature) handlePricing(w http.ResponseWriter, r *http.Request) {
	f.render(w, r, "/pricing", "pricing", func(bundle pwasti18n.Bundle) (string, templ.Component) {
		page := bundle.App.Pricing
		return page.Title, views.PricingPage(views.PricingPageData{
			Header:     views.PageHeaderData{Title: page.Title, Subtitle: page.Subtitle},
			PrimaryCTA: page.PrimaryCTA,
			CompareCTA: page.CompareCTA,
			Plans:      planOptions(page.Plans),
		})
	})
}

func (f *Feature) handlePayment(w http.ResponseWriter, r *http.Request) {
	f.render(w, r, "/payment", "payment", func(bundle pwasti18n.Bundle) (string, templ.Component) {
		page := bundle.App.Payment
		return page.Title, views.PaymentPage(views.PaymentPageData{
			Header:         views.PageHeaderData{Title: page.Title, Subtitle: page.Subtitle},
			MethodTitle:    page.MethodTitle,
			CardTitle:      page.CardTitle,
			PrimaryCTA:     page.PrimaryCTA,
			SecureHint:     page.SecureHint,
			PaymentMethods: paymentMethods(page.PaymentMethods),
		})
	})
}

func (f *Feature) handleProcessing(w http.ResponseWriter, r *http.Request) {
	f.render(w, r, "/payment/processing", "processing", func(bundle pwasti18n.Bundle) (string, templ.Component) {
		page := bundle.App.Processing
		return page.Title, views.ProcessingPage(views.ProcessingPageData{
			Header:     views.PageHeaderData{Title: page.Title, Subtitle: page.Subtitle},
			SecureHint: page.SecureHint,
		})
	})
}

func (f *Feature) handleSuccess(w http.ResponseWriter, r *http.Request) {
	f.render(w, r, "/payment/success", "success", func(bundle pwasti18n.Bundle) (string, templ.Component) {
		page := bundle.App.Success
		return page.Title, views.SuccessPage(views.SuccessPageData{
			Header:         views.PageHeaderData{Title: page.Title, Subtitle: page.Subtitle},
			PrimaryCTA:     page.PrimaryCTA,
			ManageCTA:      page.ManageCTA,
			ConfirmationNo: page.ConfirmationNo,
		})
	})
}

func (f *Feature) handleSubscription(w http.ResponseWriter, r *http.Request) {
	f.render(w, r, "/subscription", "subscription", func(bundle pwasti18n.Bundle) (string, templ.Component) {
		page := bundle.App.Subscription
		return page.Title, views.SubscriptionPage(views.SubscriptionPageData{
			Header:         views.PageHeaderData{Title: page.Title, Subtitle: page.Subtitle},
			PlanName:       page.PlanName,
			PlanMeta:       page.PlanMeta,
			Status:         page.Status,
			InactiveStatus: page.InactiveStatus,
			CancelCTA:      page.CancelCTA,
			Actions:        subscriptionActions(page.Actions),
			LocalData:      views.LocalDataTools(page.LocalData),
		})
	})
}

func (f *Feature) handleCancel(w http.ResponseWriter, r *http.Request) {
	f.render(w, r, "/subscription/cancel", "cancel", func(bundle pwasti18n.Bundle) (string, templ.Component) {
		page := bundle.App.Cancel
		return page.Title, views.CancelPage(views.CancelPageData{
			Header:       views.PageHeaderData{Title: page.Title, Subtitle: page.Subtitle},
			PrimaryCTA:   page.PrimaryCTA,
			SecondaryCTA: page.SecondaryCTA,
			Reasons:      textItems(page.Reasons),
		})
	})
}

func (f *Feature) handleOffline(w http.ResponseWriter, r *http.Request) {
	f.render(w, r, "/offline", "offline", func(bundle pwasti18n.Bundle) (string, templ.Component) {
		page := bundle.App.Offline
		return page.Title, views.OfflinePage(views.OfflinePageData{
			Header:     views.PageHeaderData{Title: page.Title, Subtitle: page.Subtitle},
			PrimaryCTA: page.PrimaryCTA,
			CachedHint: page.CachedHint,
		})
	})
}

func (f *Feature) handlePomodoro(w http.ResponseWriter, r *http.Request) {
	f.render(w, r, "/labs/pomodoro", "pomodoro", func(bundle pwasti18n.Bundle) (string, templ.Component) {
		page := bundle.App.Pomodoro
		return page.Title, views.PomodoroPage(views.PomodoroPageData{
			Header:             views.PageHeaderData{Kicker: page.Kicker, Title: page.Title, Subtitle: page.Subtitle},
			ModeWork:           page.ModeWork,
			ModeShortBreak:     page.ModeShortBreak,
			ModeLongBreak:      page.ModeLongBreak,
			StartCTA:           page.StartCTA,
			PauseCTA:           page.PauseCTA,
			ResetCTA:           page.ResetCTA,
			SkipCTA:            page.SkipCTA,
			SettingsTitle:      page.SettingsTitle,
			WorkLabel:          page.WorkLabel,
			ShortBreakLabel:    page.ShortBreakLabel,
			LongBreakLabel:     page.LongBreakLabel,
			RoundLabel:         page.RoundLabel,
			GoalLabel:          page.GoalLabel,
			SoundLabel:         page.SoundLabel,
			TickLabel:          page.TickLabel,
			NotificationsLabel: page.NotificationsLabel,
			ProgressLabel:      page.ProgressLabel,
			GoalTemplate:       page.GoalTemplate,
			SessionHint:        page.SessionHint,
			MobileTitle:        page.MobileTitle,
			MobileSubtitle:     page.MobileSubtitle,
			BackCTA:            page.BackCTA,
		})
	})
}

func (f *Feature) handleManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	http.ServeContent(w, r, "manifest.webmanifest", time.Time{}, bytes.NewReader(f.manifest))
}

func (f *Feature) handleServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, "sw.js", time.Time{}, bytes.NewReader(f.worker))
}

func (f *Feature) render(w http.ResponseWriter, r *http.Request, active string, key string, page func(pwasti18n.Bundle) (string, templ.Component)) {
	loc := locale.From(r.Context())
	bundle, err := pwasti18n.Load(loc)
	if err != nil {
		web.HandleError(w, err)
		return
	}

	title, body := page(bundle)
	layout := views.Layout(f.layoutData(r, bundle, active, title), body)
	cacheKey := "pwa:" + key + ":" + loc
	if err := web.CachedRender(r.Context(), w, r, f.cache, cacheKey, layout); err != nil {
		web.HandleError(w, err)
	}
}

func (f *Feature) layoutData(r *http.Request, bundle pwasti18n.Bundle, active string, title string) views.LayoutData {
	common := bundle.App.Common
	nav := f.merged
	if len(nav) == 0 {
		nav = f.navItems
	}
	headerNav := toAppNavItems(common.HeaderNav)
	if len(headerNav) == 0 {
		headerNav = cloneNav(f.navItems)
	}

	return views.LayoutData{
		Title:          title + " — " + fallback(common.BrandName, "FastyGo Tasks"),
		Lang:           locale.From(r.Context()),
		Active:         active,
		BrandName:      fallback(common.BrandName, "FastyGo Tasks"),
		AppName:        fallback(common.AppName, "Tasks"),
		InstallHint:    common.InstallHint,
		OpenMenuLabel:  common.OpenMenuLabel,
		CloseMenuLabel: common.CloseMenuLabel,
		NavItems:       cloneNav(nav),
		HeaderNavItems: headerNav,
		ThemeToggle: view.ThemeToggleData{
			Label:              fallback(common.Theme.Label, "Theme"),
			SwitchToDarkLabel:  fallback(common.Theme.SwitchToDarkLabel, "Switch to dark mode"),
			SwitchToLightLabel: fallback(common.Theme.SwitchToLightLabel, "Switch to light mode"),
		},
		LanguageToggle: view.BuildLanguageToggleFromContext(r.Context(),
			view.WithAvailable(common.Language.Available),
			view.WithLabel(common.Language.Label),
			view.WithCurrentLabel(common.Language.CurrentLabel),
			view.WithNextLocale(common.Language.NextLocale),
			view.WithNextLabel(common.Language.NextLabel),
			view.WithLocaleLabels(common.Language.LocaleLabels),
		),
	}
}

func cloneNav(items []app.NavItem) []app.NavItem {
	out := make([]app.NavItem, len(items))
	copy(out, items)
	return out
}

func fallback(value string, fallbackValue string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallbackValue
	}
	return value
}

func toAppNavItems(items []pwasti18n.NavLinkFixture) []app.NavItem {
	out := make([]app.NavItem, len(items))
	for i, item := range items {
		out[i] = app.NavItem{Label: item.Label, Path: item.Path, Icon: item.Icon, Order: i}
	}
	return out
}

func taskItems(items []pwasti18n.TaskFixture) []views.TaskItem {
	out := make([]views.TaskItem, len(items))
	for i, item := range items {
		out[i] = views.TaskItem(item)
	}
	return out
}

func statItems(items []pwasti18n.StatFixture) []views.StatItem {
	out := make([]views.StatItem, len(items))
	for i, item := range items {
		out[i] = views.StatItem(item)
	}
	return out
}

func textItems(items []pwasti18n.TextFixture) []views.TextItem {
	out := make([]views.TextItem, len(items))
	for i, item := range items {
		out[i] = views.TextItem(item)
	}
	return out
}

func stepItems(items []pwasti18n.StepFixture) []views.StepItem {
	out := make([]views.StepItem, len(items))
	for i, item := range items {
		out[i] = views.StepItem(item)
	}
	return out
}

func planOptions(items []pwasti18n.PlanFixture) []views.PlanOption {
	out := make([]views.PlanOption, len(items))
	for i, item := range items {
		out[i] = views.PlanOption(item)
	}
	return out
}

func paymentMethods(items []pwasti18n.PaymentMethodFixture) []views.PaymentMethod {
	out := make([]views.PaymentMethod, len(items))
	for i, item := range items {
		out[i] = views.PaymentMethod(item)
	}
	return out
}

func subscriptionActions(items []pwasti18n.SubscriptionAction) []views.SubscriptionAction {
	out := make([]views.SubscriptionAction, len(items))
	for i, item := range items {
		out[i] = views.SubscriptionAction(item)
	}
	return out
}
