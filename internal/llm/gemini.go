package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/llms/googleai"
)

// GeminiProvider wraps the langchaingo googleai adapter.
type GeminiProvider struct {
	model string
	llm   *googleai.GoogleAI
}

// NewGemini constructs a provider authenticating with apiKey.
func NewGemini(ctx context.Context, model, apiKey string) (*GeminiProvider, error) {
	if apiKey == "" {
		return nil, errors.New("GOOGLE_API_KEY is not set")
	}
	llm, err := googleai.New(ctx,
		googleai.WithAPIKey(apiKey),
		googleai.WithDefaultModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("init gemini: %w", err)
	}
	return &GeminiProvider{model: model, llm: llm}, nil
}

// Name returns "gemini:<model>".
func (p *GeminiProvider) Name() string { return JoinSpec("gemini", p.model) }

// Generate runs a JSON-mode completion against Gemini.
func (p *GeminiProvider) Generate(ctx context.Context, systemPrompt, userPrompt string, jsonSchema []byte) (string, error) {
	return generateFromModel(ctx, p.llm, systemPrompt, userPrompt, jsonSchema)
}
