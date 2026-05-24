package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/nich1/tempest-ai/internal/config"
)

// ErrProviderNotAvailable is returned (deliberately generic) when a job
// asks for a provider whose API key isn't configured. We don't leak
// which providers are configured.
var ErrProviderNotAvailable = errors.New("provider not available")

// Factory builds Providers on demand. It carries the credentials/baseURL
// for each supported backend.
type Factory struct {
	cfg config.LLM
}

// NewFactory builds the factory from typed config.
func NewFactory(cfg config.LLM) *Factory {
	return &Factory{cfg: cfg}
}

// DefaultSpec returns the configured default provider:model spec.
func (f *Factory) DefaultSpec() string {
	return JoinSpec(f.cfg.ProviderDefault, f.cfg.ModelDefault)
}

// IsAvailable reports whether the given provider has its credentials
// configured. Used by the API to validate POST /jobs early.
func (f *Factory) IsAvailable(provider string) bool {
	switch provider {
	case "ollama":
		return f.cfg.OllamaBaseURL != ""
	case "openai":
		return f.cfg.OpenAIAPIKey != ""
	case "anthropic":
		return f.cfg.AnthropicAPIKey != ""
	case "gemini":
		return f.cfg.GoogleAPIKey != ""
	default:
		return false
	}
}

// New parses the spec and constructs the corresponding Provider.
//
// Returns ErrProviderNotAvailable if the spec is well-formed but the
// matching credentials are missing - so the API can return a generic
// 400 without revealing which providers are configured.
func (f *Factory) New(ctx context.Context, spec string) (Provider, error) {
	provider, model, err := ParseSpec(spec)
	if err != nil {
		return nil, err
	}
	if !f.IsAvailable(provider) {
		return nil, ErrProviderNotAvailable
	}
	switch provider {
	case "ollama":
		return NewOllama(model, f.cfg.OllamaBaseURL)
	case "openai":
		return NewOpenAI(model, f.cfg.OpenAIAPIKey)
	case "anthropic":
		return NewAnthropic(model, f.cfg.AnthropicAPIKey)
	case "gemini":
		return NewGemini(ctx, model, f.cfg.GoogleAPIKey)
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}
