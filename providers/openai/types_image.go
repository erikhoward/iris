// providers/openai/types_image.go
package openai

// OpenAI Image API types.

// openAIImageRequest represents a request to the OpenAI image generations endpoint.
type openAIImageRequest struct {
	Model             string `json:"model"`
	Prompt            string `json:"prompt"`
	N                 int    `json:"n,omitempty"`
	Size              string `json:"size,omitempty"`
	Quality           string `json:"quality,omitempty"`
	ResponseFormat    string `json:"response_format,omitempty"` // "b64_json" or "url"
	OutputFormat      string `json:"output_format,omitempty"`   // "png", "jpeg", "webp"
	OutputCompression *int   `json:"output_compression,omitempty"`
	Background        string `json:"background,omitempty"`
	Moderation        string `json:"moderation,omitempty"`
	User              string `json:"user,omitempty"`
	Stream            bool   `json:"stream,omitempty"`
	PartialImages     int    `json:"partial_images,omitempty"`
}

// openAIImageEditRequest is handled via multipart form, not JSON.
// Fields documented here for reference during multipart encoding.
type openAIImageEditRequest struct {
	Model             string
	Prompt            string
	N                 int
	Size              string
	Quality           string
	ResponseFormat    string
	OutputFormat      string
	OutputCompression *int
	Background        string
	InputFidelity     string
	User              string
}

// openAIImageResponse represents a response from the OpenAI image API.
type openAIImageResponse struct {
	Created int64             `json:"created"`
	Data    []openAIImageData `json:"data"`
	Usage   *openAIImageUsage `json:"usage,omitempty"`
	Error   *openAIImageError `json:"error,omitempty"`
}

// openAIImageData represents a single image in the response.
type openAIImageData struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// openAIImageUsage tracks token usage for image generation.
type openAIImageUsage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

// openAIImageError represents an error from the image API.
type openAIImageError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

// openAIImageStreamEvent represents a streaming event from image generation.
type openAIImageStreamEvent struct {
	Type              string `json:"type"` // "image_generation.partial_image", "image_generation.completed"
	PartialImageIndex int    `json:"partial_image_index,omitempty"`
	B64JSON           string `json:"b64_json,omitempty"`
}

// openAIImageCompletedEvent represents the final completed image event.
type openAIImageCompletedEvent struct {
	Type         string                     `json:"type"` // "image_generation.completed"
	B64JSON      string                     `json:"b64_json"`
	CreatedAt    int64                      `json:"created_at,omitempty"`
	Size         string                     `json:"size,omitempty"`
	Quality      string                     `json:"quality,omitempty"`
	Background   string                     `json:"background,omitempty"`
	OutputFormat string                     `json:"output_format,omitempty"`
	Usage        *openAIImageCompletedUsage `json:"usage,omitempty"`
}

// openAIImageCompletedUsage contains token usage for completed image generation.
type openAIImageCompletedUsage struct {
	TotalTokens        int                           `json:"total_tokens"`
	InputTokens        int                           `json:"input_tokens"`
	OutputTokens       int                           `json:"output_tokens"`
	InputTokensDetails *openAIImageInputTokenDetails `json:"input_tokens_details,omitempty"`
}

// openAIImageInputTokenDetails breaks down input token usage.
type openAIImageInputTokenDetails struct {
	TextTokens  int `json:"text_tokens"`
	ImageTokens int `json:"image_tokens"`
}

// Responses API image generation tool types.

// responsesImageGenTool represents the image_generation tool configuration.
type responsesImageGenTool struct {
	Type           string              `json:"type"` // "image_generation"
	Quality        string              `json:"quality,omitempty"`
	Size           string              `json:"size,omitempty"`
	Background     string              `json:"background,omitempty"`
	OutputFormat   string              `json:"output_format,omitempty"`
	Compression    *int                `json:"output_compression,omitempty"`
	PartialImages  int                 `json:"partial_images,omitempty"`
	InputFidelity  string              `json:"input_fidelity,omitempty"`
	Action         string              `json:"action,omitempty"` // "auto", "generate", "edit"
	InputImageMask *responsesImageMask `json:"input_image_mask,omitempty"`
}

// responsesImageMask specifies a mask for image editing.
type responsesImageMask struct {
	FileID   string `json:"file_id,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// responsesImageGenCall represents an image_generation_call in output.
type responsesImageGenCall struct {
	ID            string `json:"id"`
	Type          string `json:"type"` // "image_generation_call"
	Status        string `json:"status"`
	Result        string `json:"result,omitempty"` // Base64 image data
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	Size          string `json:"size,omitempty"`
	Quality       string `json:"quality,omitempty"`
	Background    string `json:"background,omitempty"`
	OutputFormat  string `json:"output_format,omitempty"`
	Action        string `json:"action,omitempty"` // "generate" or "edit"
}

// responsesImageStreamEvent represents a streaming image event from Responses API.
type responsesImageStreamEvent struct {
	Type              string `json:"type"` // "response.image_generation_call.partial_image"
	PartialImageIndex int    `json:"partial_image_index"`
	PartialImageB64   string `json:"partial_image_b64"`
}
