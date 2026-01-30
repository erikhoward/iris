package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/erikhoward/iris/core"
)

func TestUploadFile(t *testing.T) {
	var initiateReceived, uploadReceived bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/upload/v1beta/files"):
			initiateReceived = true
			// Verify headers
			if r.Header.Get("X-Goog-Upload-Protocol") != "resumable" {
				t.Errorf("Missing resumable protocol header")
			}
			if r.Header.Get("X-Goog-Upload-Command") != "start" {
				t.Errorf("Missing start command header")
			}
			// Return upload URL
			w.Header().Set("X-Goog-Upload-URL", "http://"+r.Host+"/upload-target")
			w.WriteHeader(http.StatusOK)

		case strings.HasPrefix(r.URL.Path, "/upload-target"):
			uploadReceived = true
			// Verify upload headers
			if r.Header.Get("X-Goog-Upload-Command") != "upload, finalize" {
				t.Errorf("Missing finalize command header")
			}
			// Return file response
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(fileUploadResponse{
				File: File{
					Name:     "files/test-123",
					MimeType: "text/plain",
					State:    FileStateActive,
				},
			})
		}
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	file, err := provider.UploadFile(context.Background(), &FileUploadRequest{
		File:        strings.NewReader("test content"),
		DisplayName: "test.txt",
		MimeType:    "text/plain",
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}

	if !initiateReceived {
		t.Error("Initiate request not received")
	}
	if !uploadReceived {
		t.Error("Upload request not received")
	}
	if file.Name != "files/test-123" {
		t.Errorf("File.Name = %q, want files/test-123", file.Name)
	}
}

func TestGetFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/files/test-123") {
			t.Errorf("Path = %q, want suffix /files/test-123", r.URL.Path)
		}

		json.NewEncoder(w).Encode(File{
			Name:     "files/test-123",
			MimeType: "text/plain",
			State:    FileStateActive,
			URI:      "https://example.com/files/test-123",
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	file, err := provider.GetFile(context.Background(), "files/test-123")
	if err != nil {
		t.Fatalf("GetFile() error = %v", err)
	}

	if file.Name != "files/test-123" {
		t.Errorf("File.Name = %q, want files/test-123", file.Name)
	}
	if file.State != FileStateActive {
		t.Errorf("File.State = %q, want ACTIVE", file.State)
	}
}

func TestGetFile_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(geminiErrorResponse{
			Error: geminiError{
				Code:    404,
				Message: "File not found",
				Status:  "NOT_FOUND",
			},
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	_, err := provider.GetFile(context.Background(), "files/nonexistent")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	var provErr *core.ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("Expected ProviderError, got %T", err)
	}
}
