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
