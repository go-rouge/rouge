// A module of its own, like tools/goldens: a developer tool stays out of the
// library's build, dependency graph and coverage gate.
module github.com/go-rouge/rouge/tools/refbench

go 1.27.1

require (
	github.com/go-rouge/rouge v0.0.0
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/go-regexp/engine v0.4.2 // indirect
	github.com/go-ruby-regexp/regexp v0.2.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
)

replace github.com/go-rouge/rouge => ../..
