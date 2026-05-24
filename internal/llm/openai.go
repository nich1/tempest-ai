package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/llms/openai"
)

// OpenAIProvider wraps the langchaingo openai adapter.
type OpenAIProvider struct {
	model string
	llm   *openai.LLM
}

// NewOpenAI constructs a provider authenticating with apiKey.
func NewOpenAI(model, apiKey string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY is not set")
	}
	llm, err := openai.New(
		openai.WithToken(apiKey),
		openai.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("init openai: %w", err)
	}
	return &OpenAIProvider{model: model, llm: llm}, nil
}

// Name returns "openai:<model>".
func (p *OpenAIProvider) Name() string { return JoinSpec("openai", p.model) }

// Generate runs a JSON-mode completion against OpenAI.
func (p *OpenAIProvider) Generate(ctx context.Context, systemPrompt, userPrompt string, jsonSchema []byte) (string, error) {
	return generateFromModel(ctx, p.llm, systemPrompt, userPrompt, jsonSchema)
}
