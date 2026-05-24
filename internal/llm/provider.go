// Package llm wraps tmc/langchaingo with a single Provider interface.
//
// One Provider per (provider, model) pair. The factory parses the
// "<provider>:<model>" spec and returns the right adapter. JSON mode is
// requested where supported; the consumer post-validates the response
// against the user's output schema with one retry on failure.
package llm

import (
	"context"
	"errors"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// Provider is the unified interface every adapter implements.
type Provider interface {
	// Name returns the canonical "<provider>:<model>" string.
	Name() string

	// Generate runs a single non-streaming chat-style completion.
	//
	// systemPrompt may be empty.
	// userPrompt is the assembled prompt (system_prompt + prompt + inputs JSON + file content).
	// jsonSchema is the JSON Schema (Draft 2020-12) describing the desired output.
	// The adapter should return raw JSON output text (no markdown fences).
	Generate(ctx context.Context, systemPrompt, userPrompt string, jsonSchema []byte) (string, error)
}

// ParseSpec splits "<provider>:<model>" into its parts. Provider tags
// like ollama models with "llama3:8b" are handled by splitting on the
// first colon only.
func ParseSpec(spec string) (provider, model string, err error) {
	idx := strings.Index(spec, ":")
	if idx <= 0 || idx == len(spec)-1 {
		return "", "", errors.New("provider must be in form <provider>:<model>")
	}
	return strings.ToLower(strings.TrimSpace(spec[:idx])), strings.TrimSpace(spec[idx+1:]), nil
}

// JoinSpec is the inverse of ParseSpec.
func JoinSpec(provider, model string) string {
	return provider + ":" + model
}

// generateFromModel is shared logic that runs a langchaingo Model with a
// system + user prompt, JSON mode enabled, and returns the first text
// part of the response.
func generateFromModel(ctx context.Context, m llms.Model, systemPrompt, userPrompt string, jsonSchema []byte) (string, error) {
	user := userPrompt
	if len(jsonSchema) > 0 {
		user += "\n\nReturn ONLY a JSON object that conforms to this JSON Schema. Do not include markdown fences or commentary.\n\nSchema:\n" + string(jsonSchema)
	}

	messages := []llms.MessageContent{}
	if systemPrompt != "" {
		messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt))
	}
	messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, user))

	resp, err := m.GenerateContent(ctx, messages,
		llms.WithJSONMode(),
		llms.WithTemperature(0),
	)
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", errors.New("llm returned no choices")
	}
	return resp.Choices[0].Content, nil
}
