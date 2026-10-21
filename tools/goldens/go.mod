// A module of its own: `go list ./...` at the repository root must not pull a
// developer tool into the library's build, its dependency graph or its
// coverage gate. CI builds, vets and tests it separately (the `goldens` job),
// so it cannot rot unwatched.
module github.com/go-rouge/rouge/tools/goldens

go 1.27.1

require github.com/spf13/cobra v1.10.2

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
)
