package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erikhoward/iris/core"
)

func TestCreateEmbeddings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("Path = %q, want /v1/embeddings", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization header incorrect")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type header incorrect")
		}

		// Verify request body
		var req openAIEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}
		if req.Model != "text-embedding-3-small" {
			t.Errorf("Model = %q, want text-embedding-3-small", req.Model)
		}
		if len(req.Input) != 1 || req.Input[0] != "hello world" {
			t.Errorf("Input = %v, want [hello world]", req.Input)
		}

		// Return response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(openAIEmbeddingResponse{
			Object: "list",
			Data: []openAIEmbeddingData{
				{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2, 0.3}},
			},
			Model: "text-embedding-3-small",
			Usage: openAIEmbeddingUsage{PromptTokens: 2, TotalTokens: 2},
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL+"/v1"))

	resp, err := provider.CreateEmbeddings(context.Background(), &core.EmbeddingRequest{
		Model: "text-embedding-3-small",
		Input: []core.EmbeddingInput{{Text: "hello world"}},
	})
	if err != nil {
		t.Fatalf("CreateEmbeddings() error = %v", err)
	}

	if len(resp.Vectors) != 1 {
		t.Fatalf("len(Vectors) = %d, want 1", len(resp.Vectors))
	}
	if resp.Vectors[0].Index != 0 {
		t.Errorf("Vectors[0].Index = %d, want 0", resp.Vectors[0].Index)
	}
	if len(resp.Vectors[0].Vector) != 3 {
		t.Errorf("len(Vector) = %d, want 3", len(resp.Vectors[0].Vector))
	}
	if resp.Usage.PromptTokens != 2 {
		t.Errorf("Usage.PromptTokens = %d, want 2", resp.Usage.PromptTokens)
	}
}

func TestCreateEmbeddings_BatchInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIEmbeddingRequest
		json.NewDecoder(r.Body).Decode(&req)

		if len(req.Input) != 3 {
			t.Errorf("len(Input) = %d, want 3", len(req.Input))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(openAIEmbeddingResponse{
			Object: "list",
			Data: []openAIEmbeddingData{
				{Object: "embedding", Index: 0, Embedding: []float64{0.1}},
				{Object: "embedding", Index: 1, Embedding: []float64{0.2}},
				{Object: "embedding", Index: 2, Embedding: []float64{0.3}},
			},
			Model: "text-embedding-3-small",
			Usage: openAIEmbeddingUsage{PromptTokens: 6, TotalTokens: 6},
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL+"/v1"))

	resp, err := provider.CreateEmbeddings(context.Background(), &core.EmbeddingRequest{
		Model: "text-embedding-3-small",
		Input: []core.EmbeddingInput{
			{Text: "one"},
			{Text: "two"},
			{Text: "three"},
		},
	})
	if err != nil {
		t.Fatalf("CreateEmbeddings() error = %v", err)
	}

	if len(resp.Vectors) != 3 {
		t.Fatalf("len(Vectors) = %d, want 3", len(resp.Vectors))
	}

	// Verify index alignment
	for i, vec := range resp.Vectors {
		if vec.Index != i {
			t.Errorf("Vectors[%d].Index = %d, want %d", i, vec.Index, i)
		}
	}
}
