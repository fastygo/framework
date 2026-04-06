package docs

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"

	"github.com/fastygo/framework/pkg/core"
	"github.com/yuin/goldmark"
)

type DocsListQuery struct{}

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
	Slug string
}

type DocsPageResult struct {
	Slug  string
	Title string
	HTML  string
}

type DocsPageQueryHandler struct {
	pages map[string]DocsPageRender
}

type DocsPageRender struct {
	Title string
	HTML  string
}

func NewDocsPageQueryHandler(contentFS fs.FS) (*DocsPageQueryHandler, error) {
	renderedPages := map[string]DocsPageRender{}

	converter := goldmark.New()

	for _, page := range docsPages {
		contentPath := fmt.Sprintf("%s.md", page.Slug)
		rawMarkdown, err := fs.ReadFile(contentFS, contentPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read docs content %s: %w", contentPath, err)
		}

		var html bytes.Buffer
		if err := converter.Convert(rawMarkdown, &html); err != nil {
			return nil, fmt.Errorf("failed to render markdown %s: %w", page.Slug, err)
		}

		renderedPages[page.Slug] = DocsPageRender{
			Title: page.Title,
			HTML:  html.String(),
		}
	}

	return &DocsPageQueryHandler{pages: renderedPages}, nil
}

func (h *DocsPageQueryHandler) Handle(_ context.Context, query DocsPageQuery) (DocsPageResult, error) {
	rendered, ok := h.pages[query.Slug]
	if !ok {
		return DocsPageResult{}, core.NewDomainError(core.ErrorCodeNotFound, "documentation page not found")
	}

	return DocsPageResult{
		Slug:  query.Slug,
		Title: rendered.Title,
		HTML:  rendered.HTML,
	}, nil
}
