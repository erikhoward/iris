package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildFilesHeaders(t *testing.T) {
	p := New("test-key")
	headers := p.buildFilesHeaders()

	if headers.Get("x-api-key") != "test-key" {
		t.Errorf("expected x-api-key 'test-key', got %q", headers.Get("x-api-key"))
	}
	if headers.Get("anthropic-version") != DefaultVersion {
		t.Errorf("expected anthropic-version %q, got %q", DefaultVersion, headers.Get("anthropic-version"))
	}
	if headers.Get("anthropic-beta") != DefaultFilesAPIBeta {
		t.Errorf("expected anthropic-beta %q, got %q", DefaultFilesAPIBeta, headers.Get("anthropic-beta"))
	}
}

func TestBuildFilesHeadersCustomBeta(t *testing.T) {
	p := New("test-key", WithFilesAPIBeta("custom-beta-version"))
	headers := p.buildFilesHeaders()

	if headers.Get("anthropic-beta") != "custom-beta-version" {
		t.Errorf("expected anthropic-beta 'custom-beta-version', got %q", headers.Get("anthropic-beta"))
	}
}

func TestBuildFilesHeadersPreservesCustomHeaders(t *testing.T) {
	p := New("test-key", WithHeader("X-Custom", "value"))
	headers := p.buildFilesHeaders()

	if headers.Get("X-Custom") != "value" {
		t.Errorf("expected X-Custom 'value', got %q", headers.Get("X-Custom"))
	}
}

func TestUploadFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/files" {
			t.Errorf("expected /v1/files, got %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("expected multipart/form-data, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key 'test-key', got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-beta") != DefaultFilesAPIBeta {
			t.Errorf("expected anthropic-beta %q, got %s", DefaultFilesAPIBeta, r.Header.Get("anthropic-beta"))
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("failed to get file: %v", err)
		}
		defer file.Close()

		if header.Filename != "test.txt" {
			t.Errorf("expected filename 'test.txt', got %q", header.Filename)
		}

		content, _ := io.ReadAll(file)
		if string(content) != "hello world" {
			t.Errorf("expected content 'hello world', got %q", content)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(File{
			ID:           "file_011CNha8iCJcU1wXNR6q4V8w",
			Type:         "file",
			Filename:     "test.txt",
			MimeType:     "text/plain",
			SizeBytes:    11,
			CreatedAt:    "2025-04-14T12:00:00Z",
			Downloadable: false,
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	result, err := provider.UploadFile(context.Background(), &FileUploadRequest{
		File:     strings.NewReader("hello world"),
		Filename: "test.txt",
	})
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}

	if result.ID != "file_011CNha8iCJcU1wXNR6q4V8w" {
		t.Errorf("expected ID 'file_011CNha8iCJcU1wXNR6q4V8w', got %q", result.ID)
	}
	if result.Downloadable != false {
		t.Errorf("expected Downloadable false, got %v", result.Downloadable)
	}
}
