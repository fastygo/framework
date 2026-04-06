package docs

type DocsListItem struct {
	Slug  string
	Title string
}

var docsPages = []DocsListItem{
	{Slug: "quickstart", Title: "Quickstart"},
	{Slug: "developer-guide", Title: "Developer Guide"},
	{Slug: "api-reference", Title: "API Reference"},
}

func Registry() []DocsListItem {
	result := make([]DocsListItem, len(docsPages))
	copy(result, docsPages)
	return result
}
