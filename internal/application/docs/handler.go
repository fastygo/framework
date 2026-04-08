package docs

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/fastygo/framework/pkg/core"
	"github.com/yuin/goldmark"
)

type DocsListQuery struct {
	Locale string
}

type DocsListQueryResult struct {
	Pages []DocsListItem
}

type DocsListQueryHandler struct{}

func (DocsListQueryHandler) Handle(_ context.Context, _ DocsListQuery) (DocsListQueryResult, error) {
	return DocsListQueryResult{
		Pages: append([]DocsListItem(nil), docsPages...),
	}, nil
}

type DocsPageQuery struct {
	Slug   string
	Locale string
}

type DocsPageResult struct {
	Slug  string
	Title string
	HTML  string
}

type DocsPageQueryHandler struct {
	pages         map[string]map[string]DocsPageRender
	defaultLocale string
}

type DocsPageRender struct {
	Title string
	HTML  string
}

func NewDocsPageQueryHandler(contentFS fs.FS, locales []string, defaultLocale string) (*DocsPageQueryHandler, error) {
	normalizedLocales := normalizeLocales(locales)
	if len(normalizedLocales) == 0 {
		normalizedLocales = []string{"en"}
	}

	defaultLocale = normalizeLocale(defaultLocale)
	if defaultLocale == "" {
		defaultLocale = normalizedLocales[0]
	}

	if !containsLocale(normalizedLocales, defaultLocale) {
		defaultLocale = normalizedLocales[0]
	}

	// Build render cache for all configured locales.
	renderedPages := map[string]map[string]DocsPageRender{}

	converter := goldmark.New()

	for _, page := range docsPages {
		defaultContent, err := readDocsMarkdown(contentFS, defaultLocale, page.Slug)
		if err != nil {
			return nil, fmt.Errorf("failed to read default locale docs content %s/%s.md: %w", defaultLocale, page.Slug, err)
		}

		for _, locale := range normalizedLocales {
			rawMarkdown := defaultContent
			if locale != defaultLocale {
				if content, err := readDocsMarkdown(contentFS, locale, page.Slug); err == nil {
					rawMarkdown = content
				}
			}

			var html bytes.Buffer
			if err := converter.Convert(rawMarkdown, &html); err != nil {
				return nil, fmt.Errorf("failed to render markdown %s (%s): %w", page.Slug, locale, err)
			}

			localePages := renderedPages[locale]
			if localePages == nil {
				localePages = map[string]DocsPageRender{}
				renderedPages[locale] = localePages
			}

			localePages[page.Slug] = DocsPageRender{
				Title: page.Title,
				HTML:  html.String(),
			}
		}
	}

	return &DocsPageQueryHandler{
		pages:         renderedPages,
		defaultLocale: defaultLocale,
	}, nil
}

func (h *DocsPageQueryHandler) Handle(_ context.Context, query DocsPageQuery) (DocsPageResult, error) {
	locale := normalizeLocale(query.Locale)
	if locale == "" {
		locale = h.defaultLocale
	}

	localePages, ok := h.pages[locale]
	if !ok {
		locale = h.defaultLocale
		localePages, ok = h.pages[locale]
	}
	if !ok {
		return DocsPageResult{}, core.NewDomainError(core.ErrorCodeNotFound, "documentation locale not found")
	}

	rendered, ok := localePages[query.Slug]
	if !ok {
		return DocsPageResult{}, core.NewDomainError(core.ErrorCodeNotFound, "documentation page not found")
	}

	return DocsPageResult{
		Slug:  query.Slug,
		Title: rendered.Title,
		HTML:  rendered.HTML,
	}, nil
}

func readDocsMarkdown(contentFS fs.FS, locale string, slug string) ([]byte, error) {
	contentPath := fmt.Sprintf("i18n/%s/%s.md", locale, slug)
	return fs.ReadFile(contentFS, contentPath)
}

func normalizeLocale(locale string) string {
	return strings.ToLower(strings.TrimSpace(locale))
}

func containsLocale(locales []string, locale string) bool {
	for _, value := range locales {
		if value == locale {
			return true
		}
	}
	return false
}

func normalizeLocales(locales []string) []string {
	unique := make([]string, 0, len(locales))
	seen := map[string]struct{}{}
	for _, locale := range locales {
		normalized := normalizeLocale(locale)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		unique = append(unique, normalized)
	}
	return unique
}
