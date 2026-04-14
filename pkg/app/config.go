package app

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	AppBind          string
	DataSource       string
	StaticDir        string
	DefaultLocale    string
	AvailableLocales []string

	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURI  string
	SessionKey       string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		AppBind:          getEnv("APP_BIND", "127.0.0.1:8080"),
		DataSource:       getEnv("APP_DATA_SOURCE", "fixture"),
		StaticDir:        getEnv("APP_STATIC_DIR", "internal/site/web/static"),
		DefaultLocale:    getEnv("APP_DEFAULT_LOCALE", "en"),
		AvailableLocales: parseLocales(getEnv("APP_AVAILABLE_LOCALES", "en,ru")),

		OIDCIssuer:       os.Getenv("OIDC_ISSUER"),
		OIDCClientID:     os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		OIDCRedirectURI:  os.Getenv("OIDC_REDIRECT_URI"),
		SessionKey:       os.Getenv("SESSION_KEY"),
	}

	if len(cfg.AvailableLocales) == 0 {
		return Config{}, fmt.Errorf("APP_AVAILABLE_LOCALES must contain at least one locale")
	}

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func parseLocales(raw string) []string {
	parts := strings.Split(raw, ",")
	unique := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		locale := strings.TrimSpace(strings.ToLower(part))
		if locale == "" {
			continue
		}
		if _, ok := seen[locale]; ok {
			continue
		}
		seen[locale] = struct{}{}
		unique = append(unique, locale)
	}

	return unique
}
