package views

import (
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web/view"
)

type LayoutData struct {
	Title              string
	BrandName          string
	Active             string
	Lang               string
	NavItems           []app.NavItem
	CurrentEmail       string
	AccountSignOutText string
	ThemeToggle        view.ThemeToggleData
	LanguageToggle     view.LanguageToggleData
}

type OverviewData struct {
	BrandName      string
	Email          string
	ContactsCount  int
	Greeting       string
	Description    string
	ContactsLabel  string
	WorkspaceLabel string
	AccountLabel   string
	ManageText     string
	AddContactText string
	SignOutText    string
}

type ContactsPageData struct {
	Contacts             []ContactRow
	PageTitle            string
	Description          string
	AddHeader            string
	NameLabel            string
	EmailLabel           string
	CompanyLabel         string
	AddButtonText        string
	ExistingContactsText string
	NoContactsText       string
	NameHeader           string
	EmailHeader          string
	CompanyHeader        string
	CreatedHeader        string
	DeleteText           string
}

type ContactRow struct {
	ID        string
	Name      string
	Email     string
	Company   string
	CreatedAt string
}

type LoginPageData struct {
	Title         string
	Lang          string
	ReturnTo      string
	ErrorText     string
	Subtitle      string
	EmailLabel    string
	PasswordLabel string
	SubmitText    string
}
