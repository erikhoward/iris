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

func TestGenerateImageAsync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/async/images/generations":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(zaiAsyncImageResponse{
				Model:      "glm-image",
				ID:         "task-123",
				RequestID:  "req-456",
				TaskStatus: "PROCESSING",
			})
		case "/async-result/task-123":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(zaiAsyncResultResponse{
				Model:       "glm-image",
				TaskStatus:  "SUCCESS",
				ImageResult: []zaiImageResult{{URL: "https://example.com/async-image.png"}},
				RequestID:   "req-456",
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))

	resp, err := p.GenerateImageAsync(context.Background(), &core.ImageGenerateRequest{
		Model:  "glm-image",
		Prompt: "A sunset",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.TaskID != "task-123" {
		t.Errorf("TaskID = %s, want task-123", resp.TaskID)
	}
}

func TestGetImageResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/async-result/task-123" {
			t.Errorf("Path = %s, want /async-result/task-123", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zaiAsyncResultResponse{
			Model:       "glm-image",
			TaskStatus:  "SUCCESS",
			ImageResult: []zaiImageResult{{URL: "https://example.com/async-image.png"}},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))

	resp, err := p.GetImageResult(context.Background(), "task-123")
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].URL != "https://example.com/async-image.png" {
		t.Errorf("URL = %s, want https://example.com/async-image.png", resp.Data[0].URL)
	}
}
