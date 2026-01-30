package gemini

import (
	"github.com/erikhoward/iris/core"
	"github.com/erikhoward/iris/providers"
)

func init() {
	providers.Register("gemini", func(apiKey string) core.Provider {
		return New(apiKey)
	})
}
