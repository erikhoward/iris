package core

import "testing"

func TestFeatureEmbeddings_Exists(t *testing.T) {
	if FeatureEmbeddings != "embeddings" {
		t.Errorf("FeatureEmbeddings = %q, want embeddings", FeatureEmbeddings)
	}
}
