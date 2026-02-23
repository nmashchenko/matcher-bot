package embeddings

import (
	"context"
	"os"
	"testing"
)

func TestEmbed_Integration(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	c := NewClient(key)
	vec, err := c.Embed(context.Background(), "Hello, world!")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec.Slice()) != 1536 {
		t.Fatalf("expected 1536 dims, got %d", len(vec.Slice()))
	}
}
