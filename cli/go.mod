module github.com/erikhoward/iris/cli

go 1.21

require (
	github.com/spf13/cobra v1.8.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)

replace (
	github.com/erikhoward/iris/agents => ../agents
	github.com/erikhoward/iris/core => ../core
	github.com/erikhoward/iris/providers => ../providers
	github.com/erikhoward/iris/tools => ../tools
)
