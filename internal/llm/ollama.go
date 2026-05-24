package llm

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms/ollama"
)

// OllamaProvider wraps the langchaingo ollama adapter.
type OllamaProvider struct {
	model string
	llm   *ollama.LLM
}

// NewOllama constructs a provider talking to the Ollama server at baseURL.
func NewOllama(model, baseURL string) (*OllamaProvider, error) {
	llm, err := ollama.New(
		ollama.WithModel(model),
		ollama.WithServerURL(baseURL),
	)
	if err != nil {
		return nil, fmt.Errorf("init ollama: %w", err)
	}
	return &OllamaProvider{model: model, llm: llm}, nil
}

// Name returns "ollama:<model>".
func (p *OllamaProvider) Name() string { return JoinSpec("ollama", p.model) }

// Generate runs a JSON-mode completion against Ollama.
func (p *OllamaProvider) Generate(ctx context.Context, systemPrompt, userPrompt string, jsonSchema []byte) (string, error) {
	return generateFromModel(ctx, p.llm, systemPrompt, userPrompt, jsonSchema)
}
