package zai

import (
	"encoding/json"
	"testing"
)

func TestZaiImageRequestJSON(t *testing.T) {
	req := &zaiImageRequest{
		Model:   "glm-image",
		Prompt:  "A sunset",
		Quality: "hd",
		Size:    "1280x1280",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed["model"] != "glm-image" {
		t.Errorf("model = %v, want glm-image", parsed["model"])
	}
	if parsed["quality"] != "hd" {
		t.Errorf("quality = %v, want hd", parsed["quality"])
	}
}

func TestZaiImageResponseJSON(t *testing.T) {
	jsonData := `{
		"created": 1760335349,
		"data": [{"url": "https://example.com/image.png"}],
		"content_filter": [{"role": "assistant", "level": 1}]
	}`

	var resp zaiImageResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Created != 1760335349 {
		t.Errorf("Created = %d, want 1760335349", resp.Created)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].URL != "https://example.com/image.png" {
		t.Errorf("URL = %s, want https://example.com/image.png", resp.Data[0].URL)
	}
}
