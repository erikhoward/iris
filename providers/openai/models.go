// Package openai provides an OpenAI API provider implementation for Iris.
package openai

import "github.com/erikhoward/iris/core"

// Model constants for OpenAI models.
const (
	// GPT-5.2 series
	ModelGPT52        core.ModelID = "gpt-5.2"
	ModelGPT52Pro     core.ModelID = "gpt-5.2-pro"
	ModelGPT52Codex   core.ModelID = "gpt-5.2-codex"

	// GPT-5.1 series
	ModelGPT51          core.ModelID = "gpt-5.1"
	ModelGPT51Codex     core.ModelID = "gpt-5.1-codex"
	ModelGPT51CodexMini core.ModelID = "gpt-5.1-codex-mini"
	ModelGPT51CodexMax  core.ModelID = "gpt-5.1-codex-max"

	// GPT-5 series
	ModelGPT5       core.ModelID = "gpt-5"
	ModelGPT5Mini   core.ModelID = "gpt-5-mini"
	ModelGPT5Nano   core.ModelID = "gpt-5-nano"
	ModelGPT5Pro    core.ModelID = "gpt-5-pro"
	ModelGPT5Codex  core.ModelID = "gpt-5-codex"

	// GPT-4.1 series
	ModelGPT41     core.ModelID = "gpt-4.1"
	ModelGPT41Mini core.ModelID = "gpt-4.1-mini"
	ModelGPT41Nano core.ModelID = "gpt-4.1-nano"

	// GPT-4o series
	ModelGPT4o     core.ModelID = "gpt-4o"
	ModelGPT4oMini core.ModelID = "gpt-4o-mini"

	// GPT-4 series
	ModelGPT4Turbo core.ModelID = "gpt-4-turbo"
	ModelGPT4      core.ModelID = "gpt-4"

	// GPT-3.5 series
	ModelGPT35Turbo        core.ModelID = "gpt-3.5-turbo"
	ModelGPT35Turbo16k     core.ModelID = "gpt-3.5-turbo-16k"
	ModelGPT35TurboInstruct core.ModelID = "gpt-3.5-turbo-instruct"

	// Reasoning models (o-series)
	ModelO4Mini             core.ModelID = "o4-mini"
	ModelO4MiniDeepResearch core.ModelID = "o4-mini-deep-research"
	ModelO3                 core.ModelID = "o3"
	ModelO3Mini             core.ModelID = "o3-mini"
	ModelO1                 core.ModelID = "o1"
	ModelO1Pro              core.ModelID = "o1-pro"
)

// models is the static list of supported models.
var models = []core.ModelInfo{
	// GPT-5.2 series
	{
		ID:          ModelGPT52,
		DisplayName: "GPT-5.2",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT52Pro,
		DisplayName: "GPT-5.2 Pro",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT52Codex,
		DisplayName: "GPT-5.2 Codex",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	// GPT-5.1 series
	{
		ID:          ModelGPT51,
		DisplayName: "GPT-5.1",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT51Codex,
		DisplayName: "GPT-5.1 Codex",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT51CodexMini,
		DisplayName: "GPT-5.1 Codex Mini",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT51CodexMax,
		DisplayName: "GPT-5.1 Codex Max",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	// GPT-5 series
	{
		ID:          ModelGPT5,
		DisplayName: "GPT-5",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT5Mini,
		DisplayName: "GPT-5 Mini",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT5Nano,
		DisplayName: "GPT-5 Nano",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT5Pro,
		DisplayName: "GPT-5 Pro",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT5Codex,
		DisplayName: "GPT-5 Codex",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	// GPT-4.1 series
	{
		ID:          ModelGPT41,
		DisplayName: "GPT-4.1",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT41Mini,
		DisplayName: "GPT-4.1 Mini",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT41Nano,
		DisplayName: "GPT-4.1 Nano",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	// GPT-4o series
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
	// GPT-4 series
	{
		ID:          ModelGPT4Turbo,
		DisplayName: "GPT-4 Turbo",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT4,
		DisplayName: "GPT-4",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	// GPT-3.5 series
	{
		ID:          ModelGPT35Turbo,
		DisplayName: "GPT-3.5 Turbo",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT35Turbo16k,
		DisplayName: "GPT-3.5 Turbo 16k",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelGPT35TurboInstruct,
		DisplayName: "GPT-3.5 Turbo Instruct",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
		},
	},
	// Reasoning models (o-series)
	{
		ID:          ModelO4Mini,
		DisplayName: "o4-mini",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelO4MiniDeepResearch,
		DisplayName: "o4-mini Deep Research",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelO3,
		DisplayName: "o3",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelO3Mini,
		DisplayName: "o3-mini",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelO1,
		DisplayName: "o1",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
	{
		ID:          ModelO1Pro,
		DisplayName: "o1 Pro",
		Capabilities: []core.Feature{
			core.FeatureChat,
			core.FeatureChatStreaming,
			core.FeatureToolCalling,
		},
	},
}
