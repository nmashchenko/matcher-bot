package embeddings

import (
	"context"
	"fmt"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Client struct {
	api *openai.Client
}

func NewClient(apiKey string) *Client {
	c := openai.NewClient(option.WithAPIKey(apiKey))
	return &Client{api: &c}
}

func (c *Client) Embed(ctx context.Context, text string) (pgvector.Vector, error) {
	resp, err := c.api.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
		Model: openai.EmbeddingModelTextEmbedding3Small,
	})
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("openai embed: %w", err)
	}
	if len(resp.Data) == 0 {
		return pgvector.Vector{}, fmt.Errorf("openai embed: empty response")
	}

	f64 := resp.Data[0].Embedding
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return pgvector.NewVector(f32), nil
}
