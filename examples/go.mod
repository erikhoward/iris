module github.com/erikhoward/iris/examples

go 1.21

require (
	github.com/erikhoward/iris/agents v0.0.0
	github.com/erikhoward/iris/core v0.0.0
	github.com/erikhoward/iris/providers v0.0.0
	github.com/erikhoward/iris/tools v0.0.0
)

replace (
	github.com/erikhoward/iris/agents => ../agents
	github.com/erikhoward/iris/core => ../core
	github.com/erikhoward/iris/providers => ../providers
	github.com/erikhoward/iris/tools => ../tools
)
