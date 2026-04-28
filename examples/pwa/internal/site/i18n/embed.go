package i18n

import (
	"embed"

	"github.com/fastygo/framework/pkg/web/i18n"
)

//go:embed en/*.json ru/*.json
var fixtureFS embed.FS

// Locales lists the locales shipped in this example.
var Locales = []string{"en", "ru"}

type Bundle struct {
	App AppFixture
}

type AppFixture struct {
	Common       CommonFixture       `json:"common"`
	Home         HomeFixture         `json:"home"`
	Onboarding   OnboardingFixture   `json:"onboarding"`
	Paywall      PaywallFixture      `json:"paywall"`
	Pricing      PricingFixture      `json:"pricing"`
	Payment      PaymentFixture      `json:"payment"`
	Processing   ProcessingFixture   `json:"processing"`
	Success      SuccessFixture      `json:"success"`
	Subscription SubscriptionFixture `json:"subscription"`
	Cancel       CancelFixture       `json:"cancel"`
	Offline      OfflineFixture      `json:"offline"`
	Pomodoro     PomodoroFixture     `json:"pomodoro"`
}

type CommonFixture struct {
	BrandName      string           `json:"brand_name"`
	AppName        string           `json:"app_name"`
	Theme          ThemeFixture     `json:"theme"`
	Language       LangFixture      `json:"language"`
	HeaderNav      []NavLinkFixture `json:"header_nav"`
	InstallHint    string           `json:"install_hint"`
	OpenMenuLabel  string           `json:"open_menu_label"`
	CloseMenuLabel string           `json:"close_menu_label"`
}

type ThemeFixture struct {
	Label              string `json:"label"`
	SwitchToDarkLabel  string `json:"switch_to_dark_label"`
	SwitchToLightLabel string `json:"switch_to_light_label"`
}

type LangFixture struct {
	Label        string            `json:"label"`
	CurrentLabel string            `json:"current_label"`
	NextLabel    string            `json:"next_label"`
	NextLocale   string            `json:"next_locale"`
	Available    []string          `json:"available"`
	LocaleLabels map[string]string `json:"locale_labels"`
}

type NavLinkFixture struct {
	Label string `json:"label"`
	Path  string `json:"path"`
	Icon  string `json:"icon"`
}

type HomeFixture struct {
	Title        string        `json:"title"`
	Subtitle     string        `json:"subtitle"`
	Kicker       string        `json:"kicker"`
	PrimaryCTA   string        `json:"primary_cta"`
	SecondaryCTA string        `json:"secondary_cta"`
	Tasks        []TaskFixture `json:"tasks"`
	Stats        []StatFixture `json:"stats"`
	Benefits     []TextFixture `json:"benefits"`
	TaskTools    TaskTools     `json:"task_tools"`
}

type TaskFixture struct {
	Title    string `json:"title"`
	Time     string `json:"time"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

type TaskTools struct {
	Title               string `json:"title"`
	Description         string `json:"description"`
	LockedHint          string `json:"locked_hint"`
	TitleLabel          string `json:"title_label"`
	TimeLabel           string `json:"time_label"`
	StatusLabel         string `json:"status_label"`
	PriorityLabel       string `json:"priority_label"`
	SaveCTA             string `json:"save_cta"`
	ResetCTA            string `json:"reset_cta"`
	EditCTA             string `json:"edit_cta"`
	DeleteCTA           string `json:"delete_cta"`
	ActiveStatus        string `json:"active_status"`
	InactiveStatus      string `json:"inactive_status"`
	SubscriptionHint    string `json:"subscription_hint"`
	SubscriptionEditCTA string `json:"subscription_edit_cta"`
}

type StatFixture struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Hint  string `json:"hint"`
}

type TextFixture struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type OnboardingFixture struct {
	Title       string        `json:"title"`
	Subtitle    string        `json:"subtitle"`
	ContinueCTA string        `json:"continue_cta"`
	SkipCTA     string        `json:"skip_cta"`
	Steps       []StepFixture `json:"steps"`
}

type StepFixture struct {
	Label       string `json:"label"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Symbol      string `json:"symbol"`
}

type PaywallFixture struct {
	Title      string        `json:"title"`
	Subtitle   string        `json:"subtitle"`
	Badge      string        `json:"badge"`
	PrimaryCTA string        `json:"primary_cta"`
	RestoreCTA string        `json:"restore_cta"`
	Benefits   []TextFixture `json:"benefits"`
}

type PricingFixture struct {
	Title      string        `json:"title"`
	Subtitle   string        `json:"subtitle"`
	PrimaryCTA string        `json:"primary_cta"`
	CompareCTA string        `json:"compare_cta"`
	Plans      []PlanFixture `json:"plans"`
}

type PlanFixture struct {
	Name        string `json:"name"`
	Price       string `json:"price"`
	Period      string `json:"period"`
	Description string `json:"description"`
	Badge       string `json:"badge"`
	Selected    bool   `json:"selected"`
}

type PaymentFixture struct {
	Title          string                 `json:"title"`
	Subtitle       string                 `json:"subtitle"`
	MethodTitle    string                 `json:"method_title"`
	CardTitle      string                 `json:"card_title"`
	PrimaryCTA     string                 `json:"primary_cta"`
	SecureHint     string                 `json:"secure_hint"`
	PaymentMethods []PaymentMethodFixture `json:"payment_methods"`
}

type PaymentMethodFixture struct {
	Label       string `json:"label"`
	Description string `json:"description"`
	Symbol      string `json:"symbol"`
}

type ProcessingFixture struct {
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	SecureHint string `json:"secure_hint"`
}

type SuccessFixture struct {
	Title          string `json:"title"`
	Subtitle       string `json:"subtitle"`
	PrimaryCTA     string `json:"primary_cta"`
	ManageCTA      string `json:"manage_cta"`
	ConfirmationNo string `json:"confirmation_no"`
}

type SubscriptionFixture struct {
	Title          string               `json:"title"`
	Subtitle       string               `json:"subtitle"`
	PlanName       string               `json:"plan_name"`
	PlanMeta       string               `json:"plan_meta"`
	Status         string               `json:"status"`
	InactiveStatus string               `json:"inactive_status"`
	CancelCTA      string               `json:"cancel_cta"`
	Actions        []SubscriptionAction `json:"actions"`
	LocalData      LocalDataFixture     `json:"local_data"`
}

type SubscriptionAction struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

type LocalDataFixture struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ExportCTA   string `json:"export_cta"`
	ImportCTA   string `json:"import_cta"`
	ClearCTA    string `json:"clear_cta"`
	Hint        string `json:"hint"`
}

type CancelFixture struct {
	Title        string        `json:"title"`
	Subtitle     string        `json:"subtitle"`
	PrimaryCTA   string        `json:"primary_cta"`
	SecondaryCTA string        `json:"secondary_cta"`
	Reasons      []TextFixture `json:"reasons"`
}

type OfflineFixture struct {
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	PrimaryCTA string `json:"primary_cta"`
	CachedHint string `json:"cached_hint"`
}

type PomodoroFixture struct {
	Kicker             string `json:"kicker"`
	Title              string `json:"title"`
	Subtitle           string `json:"subtitle"`
	ModeWork           string `json:"mode_work"`
	ModeShortBreak     string `json:"mode_short_break"`
	ModeLongBreak      string `json:"mode_long_break"`
	StartCTA           string `json:"start_cta"`
	PauseCTA           string `json:"pause_cta"`
	ResetCTA           string `json:"reset_cta"`
	SkipCTA            string `json:"skip_cta"`
	SettingsTitle      string `json:"settings_title"`
	WorkLabel          string `json:"work_label"`
	ShortBreakLabel    string `json:"short_break_label"`
	LongBreakLabel     string `json:"long_break_label"`
	RoundLabel         string `json:"round_label"`
	GoalLabel          string `json:"goal_label"`
	SoundLabel         string `json:"sound_label"`
	TickLabel          string `json:"tick_label"`
	NotificationsLabel string `json:"notifications_label"`
	ProgressLabel      string `json:"progress_label"`
	GoalTemplate       string `json:"goal_template"`
	SessionHint        string `json:"session_hint"`
	MobileTitle        string `json:"mobile_title"`
	MobileSubtitle     string `json:"mobile_subtitle"`
	BackCTA            string `json:"back_cta"`
}

var store = i18n.New[Bundle](fixtureFS, Locales, "en", func(reader i18n.Reader, loc string) (Bundle, error) {
	app, err := i18n.DecodeSection[AppFixture](reader, loc, "app")
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{App: app}, nil
})

// Load returns the bundle for a locale, falling back to English on miss.
func Load(locale string) (Bundle, error) {
	return store.Load(locale)
}
