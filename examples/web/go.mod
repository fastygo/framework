module github.com/fastygo/framework/examples/web

go 1.25.0

require (
	github.com/a-h/templ v0.3.1001
	github.com/fastygo/blocks v0.0.0-00010101000000-000000000000
	github.com/fastygo/elements v0.0.0-00010101000000-000000000000
	github.com/fastygo/framework v0.0.0-00010101000000-000000000000
	github.com/fastygo/ui8kit v0.2.5
)

// Local development inside the framework monorepo. When this example is
// extracted into its own repository, delete the replace directive and bump
// the framework require above to a tagged release.
replace github.com/fastygo/framework => ../..

replace github.com/fastygo/blocks => ../../../Blocks

replace github.com/fastygo/elements => ../../../Elements

replace github.com/fastygo/ui8kit => ../../../@UI8Kit
