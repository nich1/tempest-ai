package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/llms/anthropic"
)

// AnthropicProvider wraps the langchaingo anthropic adapter.
type AnthropicProvider struct {
	model string
	llm   *anthropic.LLM
}

// NewAnthropic constructs a provider authenticating with apiKey.
func NewAnthropic(model, apiKey string) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, errors.New("ANTHROPIC_API_KEY is not set")
	}
	llm, err := anthropic.New(
		anthropic.WithToken(apiKey),
		anthropic.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("init anthropic: %w", err)
	}
	return &AnthropicProvider{model: model, llm: llm}, nil
}

// Name returns "anthropic:<model>".
func (p *AnthropicProvider) Name() string { return JoinSpec("anthropic", p.model) }

// Generate runs a JSON-mode completion against Anthropic.
//
// Anthropic doesn't expose a strict JSON mode the way OpenAI does, so we
// rely heavily on the schema hint appended by generateFromModel and the
// downstream schema-validation retry on the consumer.
func (p *AnthropicProvider) Generate(ctx context.Context, systemPrompt, userPrompt string, jsonSchema []byte) (string, error) {
	return generateFromModel(ctx, p.llm, systemPrompt, userPrompt, jsonSchema)
}
