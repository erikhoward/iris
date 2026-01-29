package zai

import (
	"testing"

	"github.com/erikhoward/iris/core"
)

func TestMapImageGenerateRequest(t *testing.T) {
	req := &core.ImageGenerateRequest{
		Model:   "glm-image",
		Prompt:  "A sunset over mountains",
		Size:    core.ImageSize1024x1024,
		Quality: core.ImageQualityHigh,
	}

	mapped := mapImageGenerateRequest(req)

	if mapped.Model != "glm-image" {
		t.Errorf("Model = %s, want glm-image", mapped.Model)
	}
	if mapped.Prompt != "A sunset over mountains" {
		t.Errorf("Prompt = %s, want 'A sunset over mountains'", mapped.Prompt)
	}
	if mapped.Quality != "hd" {
		t.Errorf("Quality = %s, want hd", mapped.Quality)
	}
}

func TestMapImageResponse(t *testing.T) {
	resp := &zaiImageResponse{
		Created: 1760335349,
		Data: []zaiImageData{
			{URL: "https://example.com/image.png"},
		},
	}

	mapped := mapImageResponse(resp)

	if mapped.Created != 1760335349 {
		t.Errorf("Created = %d, want 1760335349", mapped.Created)
	}
	if len(mapped.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(mapped.Data))
	}
	if mapped.Data[0].URL != "https://example.com/image.png" {
		t.Errorf("URL = %s, want https://example.com/image.png", mapped.Data[0].URL)
	}
}
