// providers/openai/client_image.go
package openai

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

// GenerateImage generates images from a text prompt using the Image API.
func (p *OpenAI) GenerateImage(ctx context.Context, req *core.ImageGenerateRequest) (*core.ImageResponse, error) {
	openaiReq := mapImageGenerateRequest(req)

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.config.BaseURL + "/images/generations"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, values := range p.buildHeaders() {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, &core.ProviderError{
			Provider: "openai",
			Code:     "network_error",
			Message:  err.Error(),
			Err:      core.ErrNetwork,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseImageError(resp)
	}

	var openaiResp openAIImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, &core.ProviderError{
			Provider: "openai",
			Code:     "decode_error",
			Message:  err.Error(),
			Err:      core.ErrDecode,
		}
	}

	if openaiResp.Error != nil {
		return nil, &core.ProviderError{
			Provider: "openai",
			Status:   resp.StatusCode,
			Code:     openaiResp.Error.Code,
			Message:  openaiResp.Error.Message,
			Err:      core.ErrBadRequest,
		}
	}

	return mapImageResponse(&openaiResp), nil
}

// EditImage edits images using the Image API edits endpoint.
func (p *OpenAI) EditImage(ctx context.Context, req *core.ImageEditRequest) (*core.ImageResponse, error) {
	if len(req.Images) == 0 {
		return nil, &core.ProviderError{
			Provider: "openai",
			Code:     "invalid_request",
			Message:  "at least one image is required",
			Err:      core.ErrBadRequest,
		}
	}

	// Create multipart form
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// Add text fields
	fields := mapImageEditRequestFields(req)
	for name, value := range fields {
		if err := w.WriteField(name, value); err != nil {
			return nil, fmt.Errorf("failed to write field %s: %w", name, err)
		}
	}

	// Add images
	for i, img := range req.Images {
		data, err := img.GetBytes()
		if err != nil {
			return nil, fmt.Errorf("failed to get image bytes: %w", err)
		}
		if data == nil {
			continue // Skip URL/FileID inputs for Image API
		}

		fieldName := "image[]"
		if i == 0 && len(req.Images) == 1 {
			fieldName = "image"
		}

		filename := img.Filename
		if filename == "" {
			filename = fmt.Sprintf("image%d.png", i)
		}

		part, err := w.CreateFormFile(fieldName, filename)
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}
		if _, err := part.Write(data); err != nil {
			return nil, fmt.Errorf("failed to write image data: %w", err)
		}
	}

	// Add mask if provided
	if req.Mask != nil {
		data, err := req.Mask.GetBytes()
		if err != nil {
			return nil, fmt.Errorf("failed to get mask bytes: %w", err)
		}
		if data != nil {
			filename := req.Mask.Filename
			if filename == "" {
				filename = "mask.png"
			}
			part, err := w.CreateFormFile("mask", filename)
			if err != nil {
				return nil, fmt.Errorf("failed to create mask file: %w", err)
			}
			if _, err := part.Write(data); err != nil {
				return nil, fmt.Errorf("failed to write mask data: %w", err)
			}
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := p.config.BaseURL + "/images/edits"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers (but override Content-Type for multipart)
	for key, values := range p.buildHeaders() {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}
	httpReq.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, &core.ProviderError{
			Provider: "openai",
			Code:     "network_error",
			Message:  err.Error(),
			Err:      core.ErrNetwork,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseImageError(resp)
	}

	var openaiResp openAIImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, &core.ProviderError{
			Provider: "openai",
			Code:     "decode_error",
			Message:  err.Error(),
			Err:      core.ErrDecode,
		}
	}

	return mapImageResponse(&openaiResp), nil
}

// parseImageError parses an error response from the image API.
func parseImageError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err != nil {
		return &core.ProviderError{
			Provider: "openai",
			Status:   resp.StatusCode,
			Code:     "unknown",
			Message:  string(body),
			Err:      mapStatusToError(resp.StatusCode),
		}
	}

	return &core.ProviderError{
		Provider: "openai",
		Status:   resp.StatusCode,
		Code:     errResp.Error.Code,
		Message:  errResp.Error.Message,
		Err:      mapStatusToError(resp.StatusCode),
	}
}

// mapStatusToError maps HTTP status codes to sentinel errors.
func mapStatusToError(status int) error {
	switch status {
	case http.StatusUnauthorized:
		return core.ErrUnauthorized
	case http.StatusTooManyRequests:
		return core.ErrRateLimited
	case http.StatusBadRequest:
		return core.ErrBadRequest
	default:
		if status >= 500 {
			return core.ErrServer
		}
		return core.ErrBadRequest
	}
}
