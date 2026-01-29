package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/erikhoward/iris/core"
)

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
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		purpose := r.FormValue("purpose")
		if purpose != "user_data" {
			t.Errorf("expected purpose 'user_data', got %q", purpose)
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
			ID:        "file-abc123",
			Object:    "file",
			Bytes:     11,
			CreatedAt: 1677610602,
			Filename:  "test.txt",
			Purpose:   FilePurposeUserData,
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL+"/v1"))

	result, err := provider.UploadFile(context.Background(), &FileUploadRequest{
		File:     strings.NewReader("hello world"),
		Filename: "test.txt",
		Purpose:  FilePurposeUserData,
	})
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}

	if result.ID != "file-abc123" {
		t.Errorf("expected ID 'file-abc123', got %q", result.ID)
	}
	if result.Purpose != FilePurposeUserData {
		t.Errorf("expected Purpose 'user_data', got %q", result.Purpose)
	}
}

func TestUploadFileWithExpiration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		anchor := r.FormValue("expires_after[anchor]")
		seconds := r.FormValue("expires_after[seconds]")

		if anchor != "created_at" {
			t.Errorf("expected anchor 'created_at', got %q", anchor)
		}
		if seconds != "2592000" {
			t.Errorf("expected seconds '2592000', got %q", seconds)
		}

		w.Header().Set("Content-Type", "application/json")
		expiresAt := int64(1680202602)
		json.NewEncoder(w).Encode(File{
			ID:        "file-abc123",
			Object:    "file",
			Bytes:     11,
			CreatedAt: 1677610602,
			ExpiresAt: &expiresAt,
			Filename:  "test.txt",
			Purpose:   FilePurposeUserData,
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL+"/v1"))

	result, err := provider.UploadFile(context.Background(), &FileUploadRequest{
		File:     strings.NewReader("hello world"),
		Filename: "test.txt",
		Purpose:  FilePurposeUserData,
		ExpiresAfter: &ExpiresAfter{
			Anchor:  "created_at",
			Seconds: 2592000,
		},
	})
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}

	if result.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	}
}

func TestUploadFileError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Invalid file purpose",
				"type":    "invalid_request_error",
				"code":    "invalid_purpose",
			},
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL+"/v1"))

	_, err := provider.UploadFile(context.Background(), &FileUploadRequest{
		File:     strings.NewReader("hello"),
		Filename: "test.txt",
		Purpose:  "invalid",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var provErr *core.ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if !errors.Is(provErr, core.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", provErr.Err)
	}
}
