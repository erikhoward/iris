// Package openai provides an OpenAI API provider implementation for Iris.
package openai

import "github.com/erikhoward/iris/core"

// Model constants for OpenAI models.
const (
	ModelGPT4o     core.ModelID = "gpt-4o"
	ModelGPT4oMini core.ModelID = "gpt-4o-mini"
)

// models is the static list of supported models.
var models = []core.ModelInfo{
	{
		ID:          ModelGPT4o,
		DisplayName: "GPT-4o",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT4oMini,
		DisplayName: "GPT-4o Mini",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
}
