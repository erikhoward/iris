package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
