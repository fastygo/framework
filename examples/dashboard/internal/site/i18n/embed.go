package i18n

import (
	"embed"

	"github.com/fastygo/framework/pkg/web/i18n"
)

//go:embed en/*.json ru/*.json
var fixtureFS embed.FS

type NavFixture struct {
	OverviewLabel string `json:"overview"`
	ContactsLabel string `json:"contacts"`
}

type ThemeFixture struct {
	Label              string `json:"label"`
	SwitchToDarkLabel  string `json:"switch_to_dark_label"`
	SwitchToLightLabel string `json:"switch_to_light_label"`
}

type CommonFixture struct {
	BrandName string     `json:"brand_name"`
	Nav       NavFixture `json:"nav"`
	Theme     ThemeFixture `json:"theme"`
	ErrorText struct {
		MissingCredentials string `json:"missing_credentials"`
	} `json:"error_text"`
}

type OverviewFixture struct {
	PageTitle        string `json:"page_title"`
	Description      string `json:"description"`
	ContactsLabel    string `json:"contacts_label"`
	WorkspaceLabel   string `json:"workspace_label"`
	AccountLabel     string `json:"account_label"`
	ManageText       string `json:"manage_text"`
	AddContactText   string `json:"add_contact_text"`
	SignOutText      string `json:"sign_out_text"`
	GreetingTemplate string `json:"greeting_template"`
}

type ContactsFixture struct {
	PageTitle            string `json:"page_title"`
	Description          string `json:"description"`
	AddHeader            string `json:"add_header"`
	NameLabel            string `json:"name_label"`
	EmailLabel           string `json:"email_label"`
	CompanyLabel         string `json:"company_label"`
	AddButtonText        string `json:"add_button_text"`
	ExistingContactsText string `json:"existing_contacts_text"`
	NoContactsText       string `json:"no_contacts_text"`
	NameHeader           string `json:"name_header"`
	EmailHeader          string `json:"email_header"`
	CompanyHeader        string `json:"company_header"`
	CreatedHeader        string `json:"created_header"`
	DeleteText           string `json:"delete_text"`
	ErrorText            struct {
		NameRequired        string `json:"name_required"`
		EmailRequired       string `json:"email_required"`
		CreateContactFailed string `json:"create_contact_failed"`
	} `json:"error_text"`
}

type LoginFixture struct {
	PageTitle        string `json:"page_title"`
	Subtitle         string `json:"subtitle"`
	EmailLabel       string `json:"email_label"`
	PasswordLabel    string `json:"password_label"`
	SubmitButtonText string `json:"submit_button_text"`
	SessionError     string `json:"session_error"`
}

type AccountFixture struct {
	SignOutText string `json:"sign_out_text"`
}

type DashboardFixture struct {
	Overview OverviewFixture `json:"overview"`
	Contacts ContactsFixture `json:"contacts"`
	Login    LoginFixture    `json:"login"`
	Account  AccountFixture  `json:"account"`
}

type Bundle struct {
	Common    CommonFixture
	Dashboard DashboardFixture
}

var store = i18n.New[Bundle](fixtureFS, []string{"en", "ru"}, "en", func(reader i18n.Reader, loc string) (Bundle, error) {
	common, err := i18n.DecodeSection[CommonFixture](reader, loc, "common")
	if err != nil {
		return Bundle{}, err
	}
	dashboard, err := i18n.DecodeSection[DashboardFixture](reader, loc, "dashboard")
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{Common: common, Dashboard: dashboard}, nil
})

func Load(locale string) (Bundle, error) {
	return store.Load(locale)
}
