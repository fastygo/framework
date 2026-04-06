package partials

type LanguageToggleData struct {
	Label         string
	CurrentLocale string
	CurrentLabel  string
	NextLocale    string
	NextLabel     string
	DefaultLocale string
	AvailableLocales []string
}
