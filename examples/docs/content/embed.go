// Package content embeds the markdown sources for the documentation site.
package content

import "embed"

//go:embed i18n/*/*.md
var FS embed.FS
