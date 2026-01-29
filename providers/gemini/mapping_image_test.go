package gemini

import (
	"testing"

	"github.com/erikhoward/iris/core"
)

func TestMapImageGenerateRequest(t *testing.T) {
	req := &core.ImageGenerateRequest{
		Model:  "gemini-2.5-flash-image-preview",
		Prompt: "A sunset over mountains",
		Size:   core.ImageSize1024x1024,
	}

	gemReq := mapImageGenerateRequest(req)

	if len(gemReq.Contents) != 1 {
		t.Fatalf("len(Contents) = %d, want 1", len(gemReq.Contents))
	}

	if gemReq.GenerationConfig == nil {
		t.Fatal("GenerationConfig is nil")
	}

	modalities := gemReq.GenerationConfig.ResponseModalities
	if len(modalities) != 2 || modalities[0] != "TEXT" || modalities[1] != "IMAGE" {
		t.Errorf("ResponseModalities = %v, want [TEXT IMAGE]", modalities)
	}
}
