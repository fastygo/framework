package welcome

import (
	"context"

	"github.com/fastygo/framework/internal/site/web/i18n"
)

type WelcomeQuery struct {
	Locale string
}

func (q WelcomeQuery) Validate() error {
	if q.Locale == "" {
		q.Locale = "en"
	}
	return nil
}

type WelcomeQueryResult struct {
	Layout  i18n.Localized
}

type WelcomeQueryHandler struct{}

func (h WelcomeQueryHandler) Handle(_ context.Context, query WelcomeQuery) (WelcomeQueryResult, error) {
	content, err := i18n.Load(query.Locale)
	if err != nil {
		return WelcomeQueryResult{}, err
	}
	return WelcomeQueryResult{Layout: content}, nil
}
