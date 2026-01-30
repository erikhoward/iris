package gemini

import (
	"encoding/json"
	"testing"
)

func TestFileState_Constants(t *testing.T) {
	if FileStateActive != "ACTIVE" {
		t.Errorf("FileStateActive = %q, want ACTIVE", FileStateActive)
	}
	if FileStateProcessing != "PROCESSING" {
		t.Errorf("FileStateProcessing = %q, want PROCESSING", FileStateProcessing)
	}
	if FileStateFailed != "FAILED" {
		t.Errorf("FileStateFailed = %q, want FAILED", FileStateFailed)
	}
}

func TestFile_JSONUnmarshal(t *testing.T) {
	input := `{
		"name": "files/abc-123",
		"displayName": "test.pdf",
		"mimeType": "application/pdf",
		"sizeBytes": "1024",
		"createTime": "2026-01-30T12:00:00Z",
		"uri": "https://example.com/files/abc-123",
		"state": "ACTIVE"
	}`

	var file File
	if err := json.Unmarshal([]byte(input), &file); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if file.Name != "files/abc-123" {
		t.Errorf("Name = %q, want files/abc-123", file.Name)
	}
	if file.MimeType != "application/pdf" {
		t.Errorf("MimeType = %q, want application/pdf", file.MimeType)
	}
	if file.State != FileStateActive {
		t.Errorf("State = %q, want ACTIVE", file.State)
	}
}

func TestFileListResponse_JSONUnmarshal(t *testing.T) {
	input := `{
		"files": [
			{"name": "files/1", "state": "ACTIVE"},
			{"name": "files/2", "state": "PROCESSING"}
		],
		"nextPageToken": "token123"
	}`

	var resp FileListResponse
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(resp.Files) != 2 {
		t.Errorf("len(Files) = %d, want 2", len(resp.Files))
	}
	if resp.NextPageToken != "token123" {
		t.Errorf("NextPageToken = %q, want token123", resp.NextPageToken)
	}
}
