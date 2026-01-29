package zai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erikhoward/iris/core"
)

func TestGenerateImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/images/generations" {
			t.Errorf("Path = %s, want /images/generations", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zaiImageResponse{
			Created: 1760335349,
			Data:    []zaiImageData{{URL: "https://example.com/image.png"}},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))

	resp, err := p.GenerateImage(context.Background(), &core.ImageGenerateRequest{
		Model:  "glm-image",
		Prompt: "A sunset",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].URL != "https://example.com/image.png" {
		t.Errorf("URL = %s, want https://example.com/image.png", resp.Data[0].URL)
	}
}

func TestGenerateImageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Invalid prompt",
				"code":    "1001",
			},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))

	_, err := p.GenerateImage(context.Background(), &core.ImageGenerateRequest{
		Model:  "glm-image",
		Prompt: "",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
