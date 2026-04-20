module github.com/fastygo/framework/examples/docs

go 1.25.0

require (
	github.com/a-h/templ v0.3.1001
	github.com/fastygo/framework v0.0.0-00010101000000-000000000000
	github.com/fastygo/ui8kit v0.2.5
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
)

// Local development inside the framework monorepo. When this example is
// extracted into its own repository, delete the replace directive and bump
// the framework require above to a tagged release.
replace github.com/fastygo/framework => ../..
