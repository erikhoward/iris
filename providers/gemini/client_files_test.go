package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
