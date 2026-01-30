package core

import "context"

// EncodingFormat specifies the embedding output format.
type EncodingFormat string

const (
	// EncodingFormatFloat returns embeddings as float arrays.
	EncodingFormatFloat EncodingFormat = "float"
	// EncodingFormatBase64 returns embeddings as base64-encoded strings.
	EncodingFormatBase64 EncodingFormat = "base64"
)

// EmbeddingInput represents a single text to embed with optional metadata.
type EmbeddingInput struct {
	Text     string            `json:"text"`
	ID       string            `json:"id,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// EmbeddingRequest represents a request to generate embeddings.
type EmbeddingRequest struct {
	Model          ModelID          `json:"model"`
	Input          []EmbeddingInput `json:"input"`
	EncodingFormat EncodingFormat   `json:"encoding_format,omitempty"`
	Dimensions     *int             `json:"dimensions,omitempty"`
	User           string           `json:"user,omitempty"`
}

// EmbeddingVector represents a single embedding result.
type EmbeddingVector struct {
	Index     int               `json:"index"`
	ID        string            `json:"id,omitempty"`
	Vector    []float32         `json:"vector,omitempty"`
	VectorB64 string            `json:"vector_b64,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// EmbeddingUsage tracks token consumption for embeddings.
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// EmbeddingResponse contains the generated embeddings.
type EmbeddingResponse struct {
	Vectors []EmbeddingVector `json:"vectors"`
	Model   ModelID           `json:"model"`
	Usage   EmbeddingUsage    `json:"usage"`
}

// EmbeddingProvider is an optional interface for providers that support embeddings.
type EmbeddingProvider interface {
	// CreateEmbeddings generates embeddings for the given input texts.
	CreateEmbeddings(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)
}
