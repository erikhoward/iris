package zai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/erikhoward/iris/core"
)

// imageGenerationsPath is the API endpoint for image generations.
const imageGenerationsPath = "/images/generations"

// GenerateImage generates images from a text prompt.
func (p *Zai) GenerateImage(ctx context.Context, req *core.ImageGenerateRequest) (*core.ImageResponse, error) {
	zaiReq := mapImageGenerateRequest(req)

	body, err := json.Marshal(zaiReq)
	if err != nil {
		return nil, newDecodeError(err)
	}

	url := p.config.BaseURL + imageGenerationsPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, newNetworkError(err)
	}

	for key, values := range p.buildHeaders() {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, newNetworkError(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newNetworkError(err)
	}

	// Extract request ID for error context
	var tempResp struct {
		RequestID string `json:"request_id"`
	}
	_ = json.Unmarshal(respBody, &tempResp)

	if resp.StatusCode >= 400 {
		return nil, normalizeError(resp.StatusCode, respBody, tempResp.RequestID)
	}

	var zaiResp zaiImageResponse
	if err := json.Unmarshal(respBody, &zaiResp); err != nil {
		return nil, newDecodeError(err)
	}

	return mapImageResponse(&zaiResp), nil
}

// EditImage is not supported by Z.ai - returns an error.
func (p *Zai) EditImage(ctx context.Context, req *core.ImageEditRequest) (*core.ImageResponse, error) {
	return nil, &core.ProviderError{
		Provider: "zai",
		Code:     "not_supported",
		Message:  "Z.ai does not support image editing",
		Err:      core.ErrBadRequest,
	}
}

// StreamImage is not supported by Z.ai - returns an error.
func (p *Zai) StreamImage(ctx context.Context, req *core.ImageGenerateRequest) (*core.ImageStream, error) {
	return nil, &core.ProviderError{
		Provider: "zai",
		Code:     "not_supported",
		Message:  "Z.ai does not support streaming image generation",
		Err:      core.ErrBadRequest,
	}
}

// asyncImageGenerationsPath is the API endpoint for async image generations.
const asyncImageGenerationsPath = "/async/images/generations"

// asyncResultPath is the API endpoint for async results.
const asyncResultPath = "/async-result"

// AsyncImageResponse represents an async image generation task.
type AsyncImageResponse struct {
	TaskID    string
	RequestID string
	Status    string
}

// GenerateImageAsync submits an async image generation request.
func (p *Zai) GenerateImageAsync(ctx context.Context, req *core.ImageGenerateRequest) (*AsyncImageResponse, error) {
	zaiReq := mapImageGenerateRequest(req)

	body, err := json.Marshal(zaiReq)
	if err != nil {
		return nil, newDecodeError(err)
	}

	url := p.config.BaseURL + asyncImageGenerationsPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, newNetworkError(err)
	}

	for key, values := range p.buildHeaders() {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, newNetworkError(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newNetworkError(err)
	}

	if resp.StatusCode >= 400 {
		return nil, normalizeError(resp.StatusCode, respBody, "")
	}

	var zaiResp zaiAsyncImageResponse
	if err := json.Unmarshal(respBody, &zaiResp); err != nil {
		return nil, newDecodeError(err)
	}

	return &AsyncImageResponse{
		TaskID:    zaiResp.ID,
		RequestID: zaiResp.RequestID,
		Status:    zaiResp.TaskStatus,
	}, nil
}

// GetImageResult retrieves the result of an async image generation.
func (p *Zai) GetImageResult(ctx context.Context, taskID string) (*core.ImageResponse, error) {
	url := p.config.BaseURL + asyncResultPath + "/" + taskID
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, newNetworkError(err)
	}

	for key, values := range p.buildHeaders() {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}

	resp, err := p.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, newNetworkError(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newNetworkError(err)
	}

	if resp.StatusCode >= 400 {
		return nil, normalizeError(resp.StatusCode, respBody, "")
	}

	var zaiResp zaiAsyncResultResponse
	if err := json.Unmarshal(respBody, &zaiResp); err != nil {
		return nil, newDecodeError(err)
	}

	// Check if task is still processing
	if zaiResp.TaskStatus == "PROCESSING" {
		return nil, &core.ProviderError{
			Provider: "zai",
			Code:     "task_processing",
			Message:  "Image generation task is still processing",
			Err:      core.ErrBadRequest,
		}
	}

	if zaiResp.TaskStatus == "FAIL" {
		return nil, &core.ProviderError{
			Provider: "zai",
			Code:     "task_failed",
			Message:  "Image generation task failed",
			Err:      core.ErrServer,
		}
	}

	return mapAsyncResultResponse(&zaiResp), nil
}
