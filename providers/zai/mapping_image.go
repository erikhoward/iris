package zai

import (
	"github.com/erikhoward/iris/core"
)

// mapImageGenerateRequest converts a core image request to Z.ai format.
func mapImageGenerateRequest(req *core.ImageGenerateRequest) *zaiImageRequest {
	r := &zaiImageRequest{
		Model:  string(req.Model),
		Prompt: req.Prompt,
	}

	// Map size - Z.ai uses different size strings
	if req.Size != "" {
		r.Size = mapImageSize(req.Size)
	}

	// Map quality
	if req.Quality != "" {
		r.Quality = mapImageQuality(req.Quality)
	}

	if req.User != "" {
		r.UserID = req.User
	}

	return r
}

// mapImageSize converts core ImageSize to Z.ai size string.
func mapImageSize(size core.ImageSize) string {
	switch size {
	case core.ImageSize1024x1024:
		return "1280x1280" // Z.ai default square
	case core.ImageSize1536x1024:
		return "1568x1056" // Approximate landscape
	case core.ImageSize1024x1536:
		return "1056x1568" // Approximate portrait
	default:
		return "1280x1280"
	}
}

// mapImageQuality converts core ImageQuality to Z.ai quality string.
func mapImageQuality(quality core.ImageQuality) string {
	switch quality {
	case core.ImageQualityHigh:
		return "hd"
	case core.ImageQualityMedium, core.ImageQualityLow:
		return "standard"
	default:
		return "hd"
	}
}

// mapImageResponse converts a Z.ai image response to core format.
func mapImageResponse(resp *zaiImageResponse) *core.ImageResponse {
	r := &core.ImageResponse{
		Created: resp.Created,
		Data:    make([]core.ImageData, len(resp.Data)),
	}

	for i, d := range resp.Data {
		r.Data[i] = core.ImageData{
			URL: d.URL,
		}
	}

	return r
}

// mapAsyncResultResponse converts an async result response to core format.
func mapAsyncResultResponse(resp *zaiAsyncResultResponse) *core.ImageResponse {
	r := &core.ImageResponse{
		Data: make([]core.ImageData, len(resp.ImageResult)),
	}

	for i, d := range resp.ImageResult {
		r.Data[i] = core.ImageData{
			URL: d.URL,
		}
	}

	return r
}
