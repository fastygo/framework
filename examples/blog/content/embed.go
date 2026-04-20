// Package content embeds blog posts as markdown files.
package content

import "embed"

//go:embed i18n/*/*.md
var FS embed.FS
