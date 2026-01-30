package core

import "testing"

func TestFeatureEmbeddings_Exists(t *testing.T) {
	if FeatureEmbeddings != "embeddings" {
		t.Errorf("FeatureEmbeddings = %q, want embeddings", FeatureEmbeddings)
	}
}

func TestEncodingFormat_Constants(t *testing.T) {
	if EncodingFormatFloat != "float" {
		t.Errorf("EncodingFormatFloat = %q, want float", EncodingFormatFloat)
	}
	if EncodingFormatBase64 != "base64" {
		t.Errorf("EncodingFormatBase64 = %q, want base64", EncodingFormatBase64)
	}
}

func TestEmbeddingRequest_Fields(t *testing.T) {
	dims := 1024
	req := EmbeddingRequest{
		Model: "text-embedding-3-small",
		Input: []EmbeddingInput{
			{Text: "hello", ID: "1", Metadata: map[string]string{"key": "val"}},
		},
		EncodingFormat: EncodingFormatFloat,
		Dimensions:     &dims,
		User:           "user-123",
	}

	if req.Model != "text-embedding-3-small" {
		t.Errorf("Model = %q, want text-embedding-3-small", req.Model)
	}
	if len(req.Input) != 1 {
		t.Errorf("len(Input) = %d, want 1", len(req.Input))
	}
	if req.Input[0].Text != "hello" {
		t.Errorf("Input[0].Text = %q, want hello", req.Input[0].Text)
	}
}

func TestEmbeddingResponse_Fields(t *testing.T) {
	resp := EmbeddingResponse{
		Vectors: []EmbeddingVector{
			{Index: 0, ID: "1", Vector: []float32{0.1, 0.2}},
		},
		Model: "text-embedding-3-small",
		Usage: EmbeddingUsage{PromptTokens: 5, TotalTokens: 5},
	}

	if len(resp.Vectors) != 1 {
		t.Errorf("len(Vectors) = %d, want 1", len(resp.Vectors))
	}
	if resp.Vectors[0].Index != 0 {
		t.Errorf("Vectors[0].Index = %d, want 0", resp.Vectors[0].Index)
	}
	if resp.Usage.PromptTokens != 5 {
		t.Errorf("Usage.PromptTokens = %d, want 5", resp.Usage.PromptTokens)
	}
}
