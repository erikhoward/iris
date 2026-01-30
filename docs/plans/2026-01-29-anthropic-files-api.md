# Anthropic Files API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Files API support to the Anthropic provider with upload, list, get, download, and delete operations.

**Architecture:** Follows existing OpenAI Files API pattern with provider-specific adaptations. Uses `anthropic-beta` header for beta API access. Download includes pre-check for `downloadable` flag.

**Tech Stack:** Go, net/http, multipart/form-data, httptest for testing

---

## Task 1: Add Sentinel Error and Update Error Handling

**Files:**
- Modify: `providers/anthropic/errors.go:11` (add sentinel)
- Modify: `providers/anthropic/errors.go:30-42` (add 404 case)
- Test: `providers/anthropic/errors_test.go`

**Step 1: Write the failing test for ErrFileNotDownloadable**

Add to `providers/anthropic/errors_test.go`:

```go
func TestErrFileNotDownloadable(t *testing.T) {
	if ErrFileNotDownloadable == nil {
		t.Error("ErrFileNotDownloadable should not be nil")
	}
	if ErrFileNotDownloadable.Error() != "file not downloadable" {
		t.Errorf("unexpected error message: %s", ErrFileNotDownloadable.Error())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./providers/anthropic -run TestErrFileNotDownloadable`
Expected: FAIL with "undefined: ErrFileNotDownloadable"

**Step 3: Add the sentinel error**

Add to `providers/anthropic/errors.go` after line 12:

```go
// ErrFileNotDownloadable is returned when attempting to download a user-uploaded file.
var ErrFileNotDownloadable = errors.New("file not downloadable")
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./providers/anthropic -run TestErrFileNotDownloadable`
Expected: PASS

**Step 5: Write the failing test for 404 handling**

Add to `providers/anthropic/errors_test.go`:

```go
func TestNormalizeError404(t *testing.T) {
	body := []byte(`{"type":"error","error":{"type":"not_found_error","message":"File not found"}}`)
	err := normalizeError(404, body, "req-123")

	var provErr *core.ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if !errors.Is(provErr, core.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", provErr.Err)
	}
}
```

**Step 6: Run test to verify it fails**

Run: `go test -v ./providers/anthropic -run TestNormalizeError404`
Expected: FAIL (currently 404 falls through to ErrServer)

**Step 7: Update normalizeError to handle 404**

Modify `providers/anthropic/errors.go` switch statement (~line 33) to add before the `case status >= 500`:

```go
	case status == http.StatusNotFound:
		sentinel = core.ErrNotFound
```

**Step 8: Run test to verify it passes**

Run: `go test -v ./providers/anthropic -run TestNormalizeError404`
Expected: PASS

**Step 9: Commit**

```bash
git add providers/anthropic/errors.go providers/anthropic/errors_test.go
git commit -m "feat(anthropic): add ErrFileNotDownloadable and 404 handling"
```

---

## Task 2: Add Configuration for Files API Beta Header

**Files:**
- Modify: `providers/anthropic/options.go` (add config field and option)
- Test: `providers/anthropic/options_test.go`

**Step 1: Write the failing test for WithFilesAPIBeta**

Add to `providers/anthropic/options_test.go`:

```go
func TestWithFilesAPIBeta(t *testing.T) {
	p := New("test-key", WithFilesAPIBeta("files-api-2025-04-14"))
	if p.config.FilesAPIBeta != "files-api-2025-04-14" {
		t.Errorf("expected FilesAPIBeta 'files-api-2025-04-14', got %q", p.config.FilesAPIBeta)
	}
}

func TestDefaultFilesAPIBeta(t *testing.T) {
	if DefaultFilesAPIBeta != "files-api-2025-04-14" {
		t.Errorf("expected default 'files-api-2025-04-14', got %q", DefaultFilesAPIBeta)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./providers/anthropic -run TestWithFilesAPIBeta`
Expected: FAIL with "undefined: WithFilesAPIBeta"

**Step 3: Add the config field and option**

Add to `providers/anthropic/options.go` after line 27 (after Timeout field):

```go
	// FilesAPIBeta is the beta version for Files API. Defaults to DefaultFilesAPIBeta.
	FilesAPIBeta string
```

Add after line 33 (DefaultVersion constant):

```go
// DefaultFilesAPIBeta is the default beta version for the Files API.
const DefaultFilesAPIBeta = "files-api-2025-04-14"
```

Add after WithTimeout function (end of file):

```go
// WithFilesAPIBeta sets the beta version header for Files API operations.
func WithFilesAPIBeta(version string) Option {
	return func(c *Config) {
		c.FilesAPIBeta = version
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test -v ./providers/anthropic -run "TestWithFilesAPIBeta|TestDefaultFilesAPIBeta"`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/anthropic/options.go providers/anthropic/options_test.go
git commit -m "feat(anthropic): add WithFilesAPIBeta configuration option"
```

---

## Task 3: Create File Types

**Files:**
- Create: `providers/anthropic/types_files.go`
- Test: `providers/anthropic/types_files_test.go`

**Step 1: Write the failing test for File type JSON unmarshaling**

Create `providers/anthropic/types_files_test.go`:

```go
package anthropic

import (
	"encoding/json"
	"testing"
)

func TestFileUnmarshal(t *testing.T) {
	data := []byte(`{
		"id": "file_011CNha8iCJcU1wXNR6q4V8w",
		"type": "file",
		"filename": "test.pdf",
		"mime_type": "application/pdf",
		"size_bytes": 1024,
		"created_at": "2025-04-14T12:00:00Z",
		"downloadable": false
	}`)

	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if f.ID != "file_011CNha8iCJcU1wXNR6q4V8w" {
		t.Errorf("expected ID 'file_011CNha8iCJcU1wXNR6q4V8w', got %q", f.ID)
	}
	if f.Type != "file" {
		t.Errorf("expected Type 'file', got %q", f.Type)
	}
	if f.Filename != "test.pdf" {
		t.Errorf("expected Filename 'test.pdf', got %q", f.Filename)
	}
	if f.MimeType != "application/pdf" {
		t.Errorf("expected MimeType 'application/pdf', got %q", f.MimeType)
	}
	if f.SizeBytes != 1024 {
		t.Errorf("expected SizeBytes 1024, got %d", f.SizeBytes)
	}
	if f.CreatedAt != "2025-04-14T12:00:00Z" {
		t.Errorf("expected CreatedAt '2025-04-14T12:00:00Z', got %q", f.CreatedAt)
	}
	if f.Downloadable != false {
		t.Errorf("expected Downloadable false, got %v", f.Downloadable)
	}
}

func TestFileListResponseUnmarshal(t *testing.T) {
	data := []byte(`{
		"data": [
			{"id": "file_1", "type": "file", "filename": "a.txt", "mime_type": "text/plain", "size_bytes": 100, "created_at": "2025-04-14T12:00:00Z", "downloadable": false}
		],
		"first_id": "file_1",
		"last_id": "file_1",
		"has_more": false
	}`)

	var resp FileListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Errorf("expected 1 file, got %d", len(resp.Data))
	}
	if resp.FirstID != "file_1" {
		t.Errorf("expected FirstID 'file_1', got %q", resp.FirstID)
	}
	if resp.HasMore != false {
		t.Errorf("expected HasMore false, got %v", resp.HasMore)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./providers/anthropic -run "TestFileUnmarshal|TestFileListResponseUnmarshal"`
Expected: FAIL with "undefined: File"

**Step 3: Create types_files.go with all types**

Create `providers/anthropic/types_files.go`:

```go
package anthropic

import "io"

// File represents an uploaded file in Anthropic.
type File struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Filename     string `json:"filename"`
	MimeType     string `json:"mime_type"`
	SizeBytes    int64  `json:"size_bytes"`
	CreatedAt    string `json:"created_at"`
	Downloadable bool   `json:"downloadable"`
}

// FileUploadRequest contains parameters for uploading a file.
type FileUploadRequest struct {
	File     io.Reader
	Filename string
	MimeType string
	Metadata map[string]string
}

// FileListRequest contains parameters for listing files.
type FileListRequest struct {
	Limit    *int
	BeforeID *string
	AfterID  *string
}

// FileListResponse contains paginated file results.
type FileListResponse struct {
	Data    []File `json:"data"`
	FirstID string `json:"first_id"`
	LastID  string `json:"last_id"`
	HasMore bool   `json:"has_more"`
}

// FileDeleteResponse contains the result of a file deletion.
type FileDeleteResponse struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}
```

**Step 4: Run tests to verify they pass**

Run: `go test -v ./providers/anthropic -run "TestFileUnmarshal|TestFileListResponseUnmarshal"`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/anthropic/types_files.go providers/anthropic/types_files_test.go
git commit -m "feat(anthropic): add file types for Files API"
```

---

## Task 4: Implement buildFilesHeaders Helper

**Files:**
- Create: `providers/anthropic/client_files.go`
- Test: `providers/anthropic/client_files_test.go`

**Step 1: Write the failing test for buildFilesHeaders**

Create `providers/anthropic/client_files_test.go`:

```go
package anthropic

import (
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
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./providers/anthropic -run TestBuildFilesHeaders`
Expected: FAIL with "p.buildFilesHeaders undefined"

**Step 3: Create client_files.go with buildFilesHeaders**

Create `providers/anthropic/client_files.go`:

```go
package anthropic

import (
	"net/http"
)

// filesPath is the API endpoint for files.
const filesPath = "/v1/files"

// buildFilesHeaders constructs headers for Files API requests.
func (p *Anthropic) buildFilesHeaders() http.Header {
	headers := p.buildHeaders()

	beta := p.config.FilesAPIBeta
	if beta == "" {
		beta = DefaultFilesAPIBeta
	}
	headers.Set("anthropic-beta", beta)

	return headers
}
```

**Step 4: Run tests to verify they pass**

Run: `go test -v ./providers/anthropic -run TestBuildFilesHeaders`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/anthropic/client_files.go providers/anthropic/client_files_test.go
git commit -m "feat(anthropic): add buildFilesHeaders helper"
```

---

## Task 5: Implement UploadFile

**Files:**
- Modify: `providers/anthropic/client_files.go`
- Modify: `providers/anthropic/client_files_test.go`

**Step 1: Write the failing test for UploadFile**

Add to `providers/anthropic/client_files_test.go`:

```go
import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./providers/anthropic -run TestUploadFile`
Expected: FAIL with "p.UploadFile undefined"

**Step 3: Implement UploadFile**

Add to `providers/anthropic/client_files.go`:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/erikhoward/iris/core"
)

// UploadFile uploads a file to Anthropic.
func (p *Anthropic) UploadFile(ctx context.Context, req *FileUploadRequest) (*File, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	part, err := w.CreateFormFile("file", req.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, req.File); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := p.config.BaseURL + filesPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, values := range p.buildFilesHeaders() {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}
	httpReq.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, newNetworkError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newNetworkError(err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, normalizeError(resp.StatusCode, body, resp.Header.Get("request-id"))
	}

	var file File
	if err := json.Unmarshal(body, &file); err != nil {
		return nil, newDecodeError(err)
	}

	return &file, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./providers/anthropic -run TestUploadFile`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/anthropic/client_files.go providers/anthropic/client_files_test.go
git commit -m "feat(anthropic): implement UploadFile"
```

---

## Task 6: Implement GetFile

**Files:**
- Modify: `providers/anthropic/client_files.go`
- Modify: `providers/anthropic/client_files_test.go`

**Step 1: Write the failing test for GetFile**

Add to `providers/anthropic/client_files_test.go`:

```go
func TestGetFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/files/file_011CNha8iCJcU1wXNR6q4V8w" {
			t.Errorf("expected /v1/files/file_011CNha8iCJcU1wXNR6q4V8w, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(File{
			ID:           "file_011CNha8iCJcU1wXNR6q4V8w",
			Type:         "file",
			Filename:     "test.pdf",
			MimeType:     "application/pdf",
			SizeBytes:    1024,
			CreatedAt:    "2025-04-14T12:00:00Z",
			Downloadable: false,
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	result, err := provider.GetFile(context.Background(), "file_011CNha8iCJcU1wXNR6q4V8w")
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}

	if result.ID != "file_011CNha8iCJcU1wXNR6q4V8w" {
		t.Errorf("expected ID 'file_011CNha8iCJcU1wXNR6q4V8w', got %q", result.ID)
	}
	if result.SizeBytes != 1024 {
		t.Errorf("expected SizeBytes 1024, got %d", result.SizeBytes)
	}
}

func TestGetFileNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    "not_found_error",
				"message": "File not found",
			},
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	_, err := provider.GetFile(context.Background(), "file-nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var provErr *core.ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if !errors.Is(provErr, core.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", provErr.Err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./providers/anthropic -run TestGetFile`
Expected: FAIL with "p.GetFile undefined"

**Step 3: Implement GetFile**

Add to `providers/anthropic/client_files.go`:

```go
// GetFile retrieves metadata for a specific file.
func (p *Anthropic) GetFile(ctx context.Context, fileID string) (*File, error) {
	url := p.config.BaseURL + filesPath + "/" + fileID

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, values := range p.buildFilesHeaders() {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, newNetworkError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newNetworkError(err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, normalizeError(resp.StatusCode, body, resp.Header.Get("request-id"))
	}

	var file File
	if err := json.Unmarshal(body, &file); err != nil {
		return nil, newDecodeError(err)
	}

	return &file, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test -v ./providers/anthropic -run "TestGetFile|TestGetFileNotFound"`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/anthropic/client_files.go providers/anthropic/client_files_test.go
git commit -m "feat(anthropic): implement GetFile"
```

---

## Task 7: Implement ListFiles

**Files:**
- Modify: `providers/anthropic/client_files.go`
- Modify: `providers/anthropic/client_files_test.go`

**Step 1: Write the failing test for ListFiles**

Add to `providers/anthropic/client_files_test.go`:

```go
func TestListFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/files" {
			t.Errorf("expected /v1/files, got %s", r.URL.Path)
		}

		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("after_id") != "file_prev" {
			t.Errorf("expected after_id=file_prev, got %s", r.URL.Query().Get("after_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(FileListResponse{
			Data: []File{
				{ID: "file_1", Type: "file", Filename: "a.txt"},
				{ID: "file_2", Type: "file", Filename: "b.txt"},
			},
			FirstID: "file_1",
			LastID:  "file_2",
			HasMore: false,
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	limit := 10
	afterID := "file_prev"
	result, err := provider.ListFiles(context.Background(), &FileListRequest{
		Limit:   &limit,
		AfterID: &afterID,
	})
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("expected 2 files, got %d", len(result.Data))
	}
	if result.Data[0].ID != "file_1" {
		t.Errorf("expected file_1, got %s", result.Data[0].ID)
	}
}

func TestListFilesNilRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(FileListResponse{
			Data: []File{},
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	result, err := provider.ListFiles(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if result.Data == nil {
		t.Error("expected empty slice, got nil")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./providers/anthropic -run TestListFiles`
Expected: FAIL with "p.ListFiles undefined"

**Step 3: Implement ListFiles**

Add to `providers/anthropic/client_files.go`:

```go
import "strconv"

// ListFiles returns a paginated list of files.
func (p *Anthropic) ListFiles(ctx context.Context, req *FileListRequest) (*FileListResponse, error) {
	url := p.config.BaseURL + filesPath

	if req != nil {
		params := make([]string, 0)
		if req.Limit != nil {
			params = append(params, "limit="+strconv.Itoa(*req.Limit))
		}
		if req.BeforeID != nil {
			params = append(params, "before_id="+*req.BeforeID)
		}
		if req.AfterID != nil {
			params = append(params, "after_id="+*req.AfterID)
		}
		if len(params) > 0 {
			url += "?" + strings.Join(params, "&")
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, values := range p.buildFilesHeaders() {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, newNetworkError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newNetworkError(err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, normalizeError(resp.StatusCode, body, resp.Header.Get("request-id"))
	}

	var result FileListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, newDecodeError(err)
	}

	return &result, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test -v ./providers/anthropic -run "TestListFiles|TestListFilesNilRequest"`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/anthropic/client_files.go providers/anthropic/client_files_test.go
git commit -m "feat(anthropic): implement ListFiles"
```

---

## Task 8: Implement ListAllFiles

**Files:**
- Modify: `providers/anthropic/client_files.go`
- Modify: `providers/anthropic/client_files_test.go`

**Step 1: Write the failing test for ListAllFiles**

Add to `providers/anthropic/client_files_test.go`:

```go
func TestListAllFiles(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			json.NewEncoder(w).Encode(FileListResponse{
				Data:    []File{{ID: "file_1"}, {ID: "file_2"}},
				FirstID: "file_1",
				LastID:  "file_2",
				HasMore: true,
			})
		} else {
			json.NewEncoder(w).Encode(FileListResponse{
				Data:    []File{{ID: "file_3"}},
				FirstID: "file_3",
				LastID:  "file_3",
				HasMore: false,
			})
		}
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	files, err := provider.ListAllFiles(context.Background())
	if err != nil {
		t.Fatalf("ListAllFiles failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./providers/anthropic -run TestListAllFiles`
Expected: FAIL with "p.ListAllFiles undefined"

**Step 3: Implement ListAllFiles**

Add to `providers/anthropic/client_files.go`:

```go
// ListAllFiles returns all files, handling pagination automatically.
func (p *Anthropic) ListAllFiles(ctx context.Context) ([]File, error) {
	var allFiles []File
	var afterID *string
	limit := 1000

	for {
		req := &FileListRequest{
			Limit:   &limit,
			AfterID: afterID,
		}

		resp, err := p.ListFiles(ctx, req)
		if err != nil {
			return nil, err
		}

		allFiles = append(allFiles, resp.Data...)

		if !resp.HasMore {
			break
		}
		afterID = &resp.LastID
	}

	return allFiles, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./providers/anthropic -run TestListAllFiles`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/anthropic/client_files.go providers/anthropic/client_files_test.go
git commit -m "feat(anthropic): implement ListAllFiles pagination helper"
```

---

## Task 9: Implement DownloadFile with Pre-check

**Files:**
- Modify: `providers/anthropic/client_files.go`
- Modify: `providers/anthropic/client_files_test.go`

**Step 1: Write the failing test for DownloadFile**

Add to `providers/anthropic/client_files_test.go`:

```go
func TestDownloadFile(t *testing.T) {
	content := []byte("file content here")
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: GetFile for pre-check
			if r.URL.Path != "/v1/files/file_downloadable" {
				t.Errorf("expected /v1/files/file_downloadable, got %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(File{
				ID:           "file_downloadable",
				Type:         "file",
				Downloadable: true,
			})
		} else {
			// Second call: actual download
			if r.URL.Path != "/v1/files/file_downloadable/content" {
				t.Errorf("expected /v1/files/file_downloadable/content, got %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(content)
		}
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	reader, err := provider.DownloadFile(context.Background(), "file_downloadable")
	if err != nil {
		t.Fatalf("DownloadFile failed: %v", err)
	}
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read content: %v", err)
	}

	if string(downloaded) != string(content) {
		t.Errorf("expected %q, got %q", content, downloaded)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls (pre-check + download), got %d", callCount)
	}
}

func TestDownloadFileNotDownloadable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(File{
			ID:           "file_uploaded",
			Type:         "file",
			Downloadable: false,
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	_, err := provider.DownloadFile(context.Background(), "file_uploaded")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrFileNotDownloadable) {
		t.Errorf("expected ErrFileNotDownloadable, got %v", err)
	}

	var provErr *core.ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if provErr.Code != "file_not_downloadable" {
		t.Errorf("expected code 'file_not_downloadable', got %q", provErr.Code)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./providers/anthropic -run TestDownloadFile`
Expected: FAIL with "p.DownloadFile undefined"

**Step 3: Implement DownloadFile**

Add to `providers/anthropic/client_files.go`:

```go
// DownloadFile retrieves the content of a file.
// Pre-checks downloadability; returns ErrFileNotDownloadable for user-uploaded files.
// Caller is responsible for closing the returned ReadCloser.
func (p *Anthropic) DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error) {
	// Pre-check: Get file metadata to verify downloadability
	file, err := p.GetFile(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if !file.Downloadable {
		return nil, &core.ProviderError{
			Provider: "anthropic",
			Code:     "file_not_downloadable",
			Message:  fmt.Sprintf("file %s is not downloadable (only tool-generated files can be downloaded)", fileID),
			Err:      ErrFileNotDownloadable,
		}
	}

	url := p.config.BaseURL + filesPath + "/" + fileID + "/content"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, values := range p.buildFilesHeaders() {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, newNetworkError(err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, normalizeError(resp.StatusCode, body, resp.Header.Get("request-id"))
	}

	return resp.Body, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test -v ./providers/anthropic -run "TestDownloadFile|TestDownloadFileNotDownloadable"`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/anthropic/client_files.go providers/anthropic/client_files_test.go
git commit -m "feat(anthropic): implement DownloadFile with pre-check"
```

---

## Task 10: Implement DeleteFile

**Files:**
- Modify: `providers/anthropic/client_files.go`
- Modify: `providers/anthropic/client_files_test.go`

**Step 1: Write the failing test for DeleteFile**

Add to `providers/anthropic/client_files_test.go`:

```go
func TestDeleteFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v1/files/file_011CNha8iCJcU1wXNR6q4V8w" {
			t.Errorf("expected /v1/files/file_011CNha8iCJcU1wXNR6q4V8w, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(FileDeleteResponse{
			ID:   "file_011CNha8iCJcU1wXNR6q4V8w",
			Type: "file_deleted",
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	err := provider.DeleteFile(context.Background(), "file_011CNha8iCJcU1wXNR6q4V8w")
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}
}

func TestDeleteFileNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    "not_found_error",
				"message": "File not found",
			},
		})
	}))
	defer server.Close()

	provider := New("test-key", WithBaseURL(server.URL))

	err := provider.DeleteFile(context.Background(), "file-nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var provErr *core.ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if !errors.Is(provErr, core.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", provErr.Err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./providers/anthropic -run TestDeleteFile`
Expected: FAIL with "p.DeleteFile undefined"

**Step 3: Implement DeleteFile**

Add to `providers/anthropic/client_files.go`:

```go
// DeleteFile deletes a file.
func (p *Anthropic) DeleteFile(ctx context.Context, fileID string) error {
	url := p.config.BaseURL + filesPath + "/" + fileID

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, values := range p.buildFilesHeaders() {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return newNetworkError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return newNetworkError(err)
	}

	if resp.StatusCode != http.StatusOK {
		return normalizeError(resp.StatusCode, body, resp.Header.Get("request-id"))
	}

	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test -v ./providers/anthropic -run "TestDeleteFile|TestDeleteFileNotFound"`
Expected: PASS

**Step 5: Commit**

```bash
git add providers/anthropic/client_files.go providers/anthropic/client_files_test.go
git commit -m "feat(anthropic): implement DeleteFile"
```

---

## Task 11: Run Full Test Suite and Final Commit

**Step 1: Run all Anthropic provider tests**

Run: `go test -v ./providers/anthropic/...`
Expected: All tests PASS

**Step 2: Run linter**

Run: `golangci-lint run ./providers/anthropic/...`
Expected: No issues

**Step 3: Final commit with all files**

```bash
git add providers/anthropic/
git commit -m "feat(anthropic): complete Files API implementation

Add full Files API support for the Anthropic provider:
- UploadFile: multipart file upload
- ListFiles: paginated listing with cursor-based pagination
- ListAllFiles: convenience helper that handles pagination
- GetFile: retrieve file metadata
- DownloadFile: download with pre-check for downloadable flag
- DeleteFile: delete files

Includes configurable beta header via WithFilesAPIBeta option
and ErrFileNotDownloadable sentinel for clear error handling."
```

**Step 4: Verify branch is ready**

Run: `git log --oneline -10`
Expected: See all commits from this implementation
