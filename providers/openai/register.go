package openai

import (
	"github.com/erikhoward/iris/core"
	"github.com/erikhoward/iris/providers"
)

func init() {
	providers.Register("openai", func(apiKey string) core.Provider {
		return New(apiKey)
	})
}
