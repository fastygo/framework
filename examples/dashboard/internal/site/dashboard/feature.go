// Package dashboard wires the protected dashboard routes (overview +
// contacts CRUD) and demonstrates how to compose an auth middleware with
// a feature.
package dashboard

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web"
	"github.com/fastygo/framework/pkg/web/locale"
	"github.com/fastygo/framework/pkg/web/view"

	"github.com/fastygo/framework/examples/dashboard/internal/domain"
	"github.com/fastygo/framework/examples/dashboard/internal/site/auth"
	"github.com/fastygo/framework/examples/dashboard/internal/site/contacts"
	dashboardi18n "github.com/fastygo/framework/examples/dashboard/internal/site/i18n"
	"github.com/fastygo/framework/examples/dashboard/internal/site/views"
)

// Feature wires the dashboard routes.
type Feature struct {
	auth     *auth.Feature
	contacts *contacts.Repository
	brand    string
}

// New returns a configured Dashboard feature.
func New(authFeature *auth.Feature, repo *contacts.Repository, brand string) *Feature {
	if brand == "" {
		brand = "Acme Dashboard"
	}
	return &Feature{auth: authFeature, contacts: repo, brand: brand}
}

func (f *Feature) ID() string { return "dashboard" }

func (f *Feature) NavItems() []app.NavItem {
	return []app.NavItem{
		{Label: "Overview", Path: "/", Icon: "home", Order: 0},
		{Label: "Contacts", Path: "/contacts", Icon: "users", Order: 1},
	}
}

func (f *Feature) NavItemsFromBundle(bundle dashboardi18n.Bundle) []app.NavItem {
	return []app.NavItem{
		{Label: fallback(bundle.Common.Nav.OverviewLabel, "Overview"), Path: "/", Icon: "home", Order: 0},
		{Label: fallback(bundle.Common.Nav.ContactsLabel, "Contacts"), Path: "/contacts", Icon: "users", Order: 1},
	}
}

func (f *Feature) localeBundle(loc string) dashboardi18n.Bundle {
	bundle, err := dashboardi18n.Load(loc)
	if err == nil {
		return bundle
	}
	return fallbackBundle()
}

func (f *Feature) Routes(mux *http.ServeMux) {
	protect := f.auth.Middleware

	mux.Handle("GET /{$}", protect(http.HandlerFunc(f.handleOverview)))
	mux.Handle("GET /contacts", protect(http.HandlerFunc(f.handleContactsList)))
	mux.Handle("POST /contacts", protect(http.HandlerFunc(f.handleContactsCreate)))
	mux.Handle("POST /contacts/{id}/delete", protect(http.HandlerFunc(f.handleContactsDelete)))
}

func (f *Feature) handleOverview(w http.ResponseWriter, r *http.Request) {
	user := f.auth.CurrentUser(r)
	contactsCount := len(f.contacts.List())
	loc := locale.From(r.Context())
	bundle := f.localeBundle(loc)

	brandName := f.brand
	if bundle.Common.BrandName != "" {
		brandName = bundle.Common.BrandName
	}

	overview := bundle.Dashboard.Overview
	signOutText := fallback(bundle.Dashboard.Account.SignOutText, overview.SignOutText, "Sign out")

	layout := views.LayoutData{
		Title:              fallback(overview.PageTitle, "Overview"),
		BrandName:          brandName,
		Active:             "/",
		NavItems:           f.NavItemsFromBundle(bundle),
		CurrentEmail:       user,
		Lang:               loc,
		AccountSignOutText: signOutText,
		ThemeToggle:        themedToggle(bundle.Common.Theme),
		LanguageToggle:     view.BuildLanguageToggleFromContext(r.Context()),
	}
	if err := web.Render(
		r.Context(),
		w,
		views.DashboardLayout(layout, views.OverviewPage(views.OverviewData{
			BrandName:      brandName,
			Email:          user,
			ContactsCount:  contactsCount,
			Greeting:       fmt.Sprintf(fallback(overview.GreetingTemplate, "Welcome back, %s"), user),
			Description:    fmt.Sprintf(fallback(overview.Description, "A quick look at what's happening in your %s workspace."), brandName),
			ContactsLabel:  fallback(overview.ContactsLabel, "Contacts"),
			WorkspaceLabel: fallback(overview.WorkspaceLabel, "Workspace"),
			AccountLabel:   fallback(overview.AccountLabel, "Account"),
			ManageText:     fallback(overview.ManageText, "Manage →"),
			AddContactText: fallback(overview.AddContactText, "Add a contact"),
			SignOutText:    signOutText,
		})),
	); err != nil {
		web.HandleError(w, err)
	}
}

func (f *Feature) handleContactsList(w http.ResponseWriter, r *http.Request) {
	user := f.auth.CurrentUser(r)
	rows := toContactRows(f.contacts.List())
	loc := locale.From(r.Context())
	bundle := f.localeBundle(loc)

	brandName := f.brand
	if bundle.Common.BrandName != "" {
		brandName = bundle.Common.BrandName
	}

	contactsText := bundle.Dashboard.Contacts
	layout := views.LayoutData{
		Title:              fallback(contactsText.PageTitle, "Contacts"),
		BrandName:          brandName,
		Active:             "/contacts",
		NavItems:           f.NavItemsFromBundle(bundle),
		CurrentEmail:       user,
		Lang:               loc,
		AccountSignOutText: fallback(bundle.Dashboard.Account.SignOutText, "Sign out"),
		ThemeToggle:        themedToggle(bundle.Common.Theme),
		LanguageToggle:     view.BuildLanguageToggleFromContext(r.Context()),
	}
	if err := web.Render(
		r.Context(),
		w,
		views.DashboardLayout(layout, views.ContactsPage(views.ContactsPageData{
			PageTitle:            fallback(contactsText.PageTitle, "Contacts"),
			Description:          fallback(contactsText.Description, "A starter CRUD scaffold backed by an in-memory repository."),
			AddHeader:            fallback(contactsText.AddHeader, "Add a contact"),
			NameLabel:            fallback(contactsText.NameLabel, "Name"),
			EmailLabel:           fallback(contactsText.EmailLabel, "Email"),
			CompanyLabel:         fallback(contactsText.CompanyLabel, "Company"),
			AddButtonText:        fallback(contactsText.AddButtonText, "Add"),
			ExistingContactsText: fallback(contactsText.ExistingContactsText, "Existing contacts"),
			NoContactsText:       fallback(contactsText.NoContactsText, "No contacts yet."),
			NameHeader:           fallback(contactsText.NameHeader, "Name"),
			EmailHeader:          fallback(contactsText.EmailHeader, "Email"),
			CompanyHeader:        fallback(contactsText.CompanyHeader, "Company"),
			CreatedHeader:        fallback(contactsText.CreatedHeader, "Created"),
			DeleteText:           fallback(contactsText.DeleteText, "Delete"),
			Contacts:             rows,
		})),
	); err != nil {
		web.HandleError(w, err)
	}
}

func (f *Feature) handleContactsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data.", http.StatusBadRequest)
		return
	}
	contact := domain.Contact{
		Name:    strings.TrimSpace(r.PostForm.Get("name")),
		Email:   strings.TrimSpace(r.PostForm.Get("email")),
		Company: strings.TrimSpace(r.PostForm.Get("company")),
	}
	_, err := f.contacts.Create(contact)
	if err != nil {
		bundle := f.localeBundle(locale.From(r.Context()))
		contactsText := bundle.Dashboard.Contacts
		message := fallback(contactsText.ErrorText.CreateContactFailed, "Unable to add contact.")
		switch {
		case errors.Is(err, domain.ErrContactNameRequired):
			message = fallback(contactsText.ErrorText.NameRequired, "Contact name is required.")
		case errors.Is(err, domain.ErrContactEmailRequired):
			message = fallback(contactsText.ErrorText.EmailRequired, "Contact email is required.")
		}
		http.Error(w, message, http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/contacts", http.StatusSeeOther)
}

func (f *Feature) handleContactsDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	f.contacts.Delete(id)
	http.Redirect(w, r, "/contacts", http.StatusSeeOther)
}

func fallbackBundle() dashboardi18n.Bundle {
	bundle, _ := dashboardi18n.Load("en")
	return bundle
}

func themedToggle(theme dashboardi18n.ThemeFixture) view.ThemeToggleData {
	return view.ThemeToggleData{
		Label:              fallback(theme.Label, "Theme"),
		SwitchToDarkLabel:  fallback(theme.SwitchToDarkLabel, "Switch to dark mode"),
		SwitchToLightLabel: fallback(theme.SwitchToLightLabel, "Switch to light mode"),
	}
}

func fallback(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func toContactRows(items []domain.Contact) []views.ContactRow {
	rows := make([]views.ContactRow, 0, len(items))
	for _, c := range items {
		rows = append(rows, views.ContactRow{
			ID:        c.ID,
			Name:      c.Name,
			Email:     c.Email,
			Company:   c.Company,
			CreatedAt: c.CreatedAt.Format("2006-01-02"),
		})
	}
	return rows
}
