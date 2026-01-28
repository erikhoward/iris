package gemini

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/erikhoward/iris/core"
	"github.com/erikhoward/iris/tools"
)

// schemaProvider is an interface for tools that provide a JSON schema.
type schemaProvider interface {
	Schema() tools.ToolSchema
}

// buildRequest creates a Gemini API request from an Iris ChatRequest.
func buildRequest(req *core.ChatRequest) *geminiRequest {
	system, contents := mapMessages(req.Messages)

	gemReq := &geminiRequest{
		Contents: contents,
	}

	// Set system instruction if present
	if system != "" {
		gemReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: system}},
		}
	}

	// Build generation config
	genConfig := &geminiGenConfig{}
	hasGenConfig := false

	if req.Temperature != nil {
		genConfig.Temperature = req.Temperature
		hasGenConfig = true
	}

	if req.MaxTokens != nil {
		genConfig.MaxOutputTokens = req.MaxTokens
		hasGenConfig = true
	}

	// Build thinking config if reasoning effort specified
	if req.ReasoningEffort != "" {
		genConfig.ThinkingConfig = buildThinkingConfig(string(req.Model), req.ReasoningEffort)
		if genConfig.ThinkingConfig != nil {
			hasGenConfig = true
		}
	}

	if hasGenConfig {
		gemReq.GenerationConfig = genConfig
	}

	// Map tools if present
	if len(req.Tools) > 0 {
		gemReq.Tools = mapTools(req.Tools)
	}

	return gemReq
}

// mapMessages converts Iris messages to Gemini format.
// It extracts system messages into a single string and converts
// user/assistant messages to the Gemini content format.
func mapMessages(msgs []core.Message) (system string, contents []geminiContent) {
	var systemParts []string

	for _, msg := range msgs {
		switch msg.Role {
		case core.RoleSystem:
			systemParts = append(systemParts, msg.Content)
		case core.RoleUser:
			contents = append(contents, geminiContent{
				Role:  "user",
				Parts: []geminiPart{{Text: msg.Content}},
			})
		case core.RoleAssistant:
			contents = append(contents, geminiContent{
				Role:  "model",
				Parts: []geminiPart{{Text: msg.Content}},
			})
		}
	}

	// Concatenate system messages with double newlines
	if len(systemParts) > 0 {
		system = strings.Join(systemParts, "\n\n")
	}

	return system, contents
}

// buildThinkingConfig creates thinking configuration based on model and effort.
func buildThinkingConfig(model string, effort core.ReasoningEffort) *geminiThinkConfig {
	if effort == "" {
		return nil
	}

	cfg := &geminiThinkConfig{
		IncludeThoughts: true,
	}

	if isGemini3Model(model) {
		// Gemini 3 uses thinkingLevel
		cfg.ThinkingLevel = mapThinkingLevel(model, effort)
	} else {
		// Gemini 2.5 uses thinkingBudget
		budget := mapThinkingBudget(effort)
		cfg.ThinkingBudget = &budget
	}

	return cfg
}

// mapThinkingLevel maps ReasoningEffort to Gemini 3 thinkingLevel.
func mapThinkingLevel(model string, effort core.ReasoningEffort) string {
	// Gemini 3 Pro cannot disable thinking
	if model == string(ModelGemini3Pro) && effort == core.ReasoningEffortNone {
		return "low"
	}

	switch effort {
	case core.ReasoningEffortNone:
		return "minimal"
	case core.ReasoningEffortLow:
		return "low"
	case core.ReasoningEffortMedium:
		return "medium"
	case core.ReasoningEffortHigh, core.ReasoningEffortXHigh:
		return "high"
	default:
		return "medium"
	}
}

// mapThinkingBudget maps ReasoningEffort to Gemini 2.5 thinkingBudget.
func mapThinkingBudget(effort core.ReasoningEffort) int {
	switch effort {
	case core.ReasoningEffortNone:
		return 0
	case core.ReasoningEffortLow:
		return 1024
	case core.ReasoningEffortMedium:
		return 8192
	case core.ReasoningEffortHigh:
		return 24576
	case core.ReasoningEffortXHigh:
		return 32768
	default:
		return 8192
	}
}

// mapTools converts Iris tools to Gemini tool format.
func mapTools(irisTools []core.Tool) []geminiTool {
	if len(irisTools) == 0 {
		return nil
	}

	decls := make([]geminiFunctionDecl, len(irisTools))
	for i, t := range irisTools {
		var params json.RawMessage

		// Check if the tool provides a schema
		if sp, ok := t.(schemaProvider); ok {
			params = sp.Schema().JSONSchema
		}

		// Default to empty object if no schema
		if params == nil {
			params = json.RawMessage(`{}`)
		}

		decls[i] = geminiFunctionDecl{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  params,
		}
	}

	return []geminiTool{{FunctionDeclarations: decls}}
}

// mapResponse converts a Gemini response to an Iris ChatResponse.
func mapResponse(resp *geminiResponse, model string) (*core.ChatResponse, error) {
	result := &core.ChatResponse{
		Model: core.ModelID(model),
	}

	// Map usage
	if resp.UsageMetadata != nil {
		result.Usage = core.TokenUsage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.PromptTokenCount + resp.UsageMetadata.CandidatesTokenCount,
		}
	}

	// Extract content from first candidate
	if len(resp.Candidates) == 0 {
		return result, nil
	}

	candidate := resp.Candidates[0]

	var textParts []string
	var toolCalls []core.ToolCall
	var thoughtParts []string
	toolCallIndex := 0

	for _, part := range candidate.Content.Parts {
		// Check if this is a thought part
		if part.Thought != nil && *part.Thought {
			if part.Text != "" {
				thoughtParts = append(thoughtParts, part.Text)
			}
			continue
		}

		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}

		if part.FunctionCall != nil {
			toolCalls = append(toolCalls, core.ToolCall{
				ID:        fmt.Sprintf("call_%d", toolCallIndex),
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
			})
			toolCallIndex++
		}
	}

	result.Output = strings.Join(textParts, "")
	result.ToolCalls = toolCalls

	// Add reasoning output if thoughts were present
	if len(thoughtParts) > 0 {
		result.Reasoning = &core.ReasoningOutput{
			Summary: thoughtParts,
		}
	}

	return result, nil
}
