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

func TestBuildFilesHeadersPreservesCustomHeaders(t *testing.T) {
	p := New("test-key", WithHeader("X-Custom", "value"))
	headers := p.buildFilesHeaders()

	if headers.Get("X-Custom") != "value" {
		t.Errorf("expected X-Custom 'value', got %q", headers.Get("X-Custom"))
	}
}
