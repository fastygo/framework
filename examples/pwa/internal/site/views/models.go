package views

import (
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web/view"
)

type LayoutData struct {
	Title          string
	Lang           string
	Active         string
	BrandName      string
	AppName        string
	InstallHint    string
	OpenMenuLabel  string
	CloseMenuLabel string
	NavItems       []app.NavItem
	HeaderNavItems []app.NavItem
	ThemeToggle    view.ThemeToggleData
	LanguageToggle view.LanguageToggleData
}

type PageHeaderData struct {
	Kicker   string
	Title    string
	Subtitle string
}

type HomePageData struct {
	Header       PageHeaderData
	PrimaryCTA   string
	SecondaryCTA string
	Tasks        []TaskItem
	Stats        []StatItem
	Benefits     []TextItem
	TaskTools    TaskTools
}

type TaskItem struct {
	Title    string
	Time     string
	Status   string
	Priority string
}

type TaskTools struct {
	Title               string
	Description         string
	LockedHint          string
	TitleLabel          string
	TimeLabel           string
	StatusLabel         string
	PriorityLabel       string
	SaveCTA             string
	ResetCTA            string
	EditCTA             string
	DeleteCTA           string
	ActiveStatus        string
	InactiveStatus      string
	SubscriptionHint    string
	SubscriptionEditCTA string
}

type StatItem struct {
	Label string
	Value string
	Hint  string
}

type TextItem struct {
	Title       string
	Description string
}

type OnboardingPageData struct {
	Header      PageHeaderData
	ContinueCTA string
	SkipCTA     string
	Steps       []StepItem
}

type StepItem struct {
	Label       string
	Title       string
	Description string
	Symbol      string
}

type PaywallPageData struct {
	Header     PageHeaderData
	Badge      string
	PrimaryCTA string
	RestoreCTA string
	Benefits   []TextItem
}

type PricingPageData struct {
	Header     PageHeaderData
	PrimaryCTA string
	CompareCTA string
	Plans      []PlanOption
}

type PlanOption struct {
	Name        string
	Price       string
	Period      string
	Description string
	Badge       string
	Selected    bool
}

type PaymentPageData struct {
	Header         PageHeaderData
	MethodTitle    string
	CardTitle      string
	PrimaryCTA     string
	SecureHint     string
	PaymentMethods []PaymentMethod
}

type PaymentMethod struct {
	Label       string
	Description string
	Symbol      string
}

type ProcessingPageData struct {
	Header     PageHeaderData
	SecureHint string
}

type SuccessPageData struct {
	Header         PageHeaderData
	PrimaryCTA     string
	ManageCTA      string
	ConfirmationNo string
}

type SubscriptionPageData struct {
	Header         PageHeaderData
	PlanName       string
	PlanMeta       string
	Status         string
	InactiveStatus string
	CancelCTA      string
	Actions        []SubscriptionAction
	LocalData      LocalDataTools
}

type SubscriptionAction struct {
	Label       string
	Description string
}

type LocalDataTools struct {
	Title       string
	Description string
	ExportCTA   string
	ImportCTA   string
	ClearCTA    string
	Hint        string
}

type CancelPageData struct {
	Header       PageHeaderData
	PrimaryCTA   string
	SecondaryCTA string
	Reasons      []TextItem
}

type OfflinePageData struct {
	Header     PageHeaderData
	PrimaryCTA string
	CachedHint string
}

type PomodoroPageData struct {
	Header             PageHeaderData
	ModeWork           string
	ModeShortBreak     string
	ModeLongBreak      string
	StartCTA           string
	PauseCTA           string
	ResetCTA           string
	SkipCTA            string
	SettingsTitle      string
	WorkLabel          string
	ShortBreakLabel    string
	LongBreakLabel     string
	RoundLabel         string
	GoalLabel          string
	SoundLabel         string
	TickLabel          string
	NotificationsLabel string
	ProgressLabel      string
	GoalTemplate       string
	SessionHint        string
	MobileTitle        string
	MobileSubtitle     string
	BackCTA            string
}
