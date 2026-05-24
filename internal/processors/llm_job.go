// Package processors holds the Asynq task handlers.
//
// The single task type today is TypeProcessJob (see internal/queue/tasks.go),
// implemented by LLMJobProcessor. It:
//
//  1. Loads the job row.
//  2. Marks it PROCESSING (and increments attempt).
//  3. Builds the user prompt from prompt + inputs JSON + (optional) file content.
//  4. Calls the LLM with JSON mode + the output JSON Schema.
//  5. Validates the LLM response against the user-supplied output schema.
//     One retry is allowed per call: if the parsed JSON doesn't match,
//     we re-prompt with the validation error message.
//  6. On success, marks the job COMPLETED with the validated output.
//     On failure, marks it FAILED with a user-visible error.
//
// All steps emit structured log lines under the request_id propagated
// from the API via the Asynq task payload.
package processors

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/hibiken/asynq"

	"github.com/nich1/tempest-ai/internal/config"
	"github.com/nich1/tempest-ai/internal/db/sqlc"
	"github.com/nich1/tempest-ai/internal/jobs"
	"github.com/nich1/tempest-ai/internal/llm"
	"github.com/nich1/tempest-ai/internal/logging"
	"github.com/nich1/tempest-ai/internal/queue"
	"github.com/nich1/tempest-ai/internal/schema"
	"github.com/nich1/tempest-ai/internal/storage"
)

// LLMJobProcessor wires the dependencies needed to run a job.
type LLMJobProcessor struct {
	cfg     config.ConsumerConfig
	jobs    *jobs.Repository
	storage *storage.Client
	factory *llm.Factory
}

// NewLLMJobProcessor builds the processor.
func NewLLMJobProcessor(
	cfg config.ConsumerConfig,
	jobsRepo *jobs.Repository,
	store *storage.Client,
	factory *llm.Factory,
) *LLMJobProcessor {
	return &LLMJobProcessor{cfg: cfg, jobs: jobsRepo, storage: store, factory: factory}
}

// Handle is the asynq.HandlerFunc target. It returns an error when the
// task should be retried (Asynq enforces MaxRetry / backoff itself).
//
// We deliberately don't return errors for "user-data" failures (bad
// schema, validation error). Those terminate the job in FAILED status so
// they don't waste retries.
func (p *LLMJobProcessor) Handle(ctx context.Context, t *asynq.Task) error {
	var payload queue.ProcessJobPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	logger := logging.FromContext(ctx)
	jobID := payload.JobID

	row, err := p.jobs.GetByID(ctx, jobID)
	if err != nil {
		if errors.Is(err, jobs.ErrNotFound) {
			logger.Warn("job.not_found", slog.String("job_id", jobID.String()))
			return nil
		}
		return fmt.Errorf("load job: %w", err)
	}

	if _, err := p.jobs.MarkProcessing(ctx, jobID); err != nil {
		return fmt.Errorf("mark processing: %w", err)
	}
	logger.Info("job.processing")

	output, runErr := p.run(ctx, row)
	if runErr != nil {
		errMsg := truncate(runErr.Error(), 1000)
		if _, err := p.jobs.MarkFailed(ctx, jobID, errMsg); err != nil {
			logger.Error("job.mark_failed_db_error", slog.Any("error", err))
		}
		logger.Warn("job.failed", slog.String("reason", errMsg))
		return nil
	}

	if _, err := p.jobs.MarkCompleted(ctx, jobID, output); err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}
	logger.Info("job.completed", slog.Int("output_bytes", len(output)))
	return nil
}

// run is the actual job pipeline.
func (p *LLMJobProcessor) run(ctx context.Context, j sqlc.Job) (json.RawMessage, error) {
	logger := logging.FromContext(ctx)

	inputSchema, err := schema.Parse(j.InputSchema)
	if err != nil {
		return nil, fmt.Errorf("input_schema parse: %w", err)
	}
	if err := inputSchema.Validate(schema.KindInput); err != nil {
		return nil, fmt.Errorf("input_schema invalid: %w", err)
	}
	outputSchema, err := schema.Parse(j.OutputSchema)
	if err != nil {
		return nil, fmt.Errorf("output_schema parse: %w", err)
	}
	if err := outputSchema.Validate(schema.KindOutput); err != nil {
		return nil, fmt.Errorf("output_schema invalid: %w", err)
	}
	if err := inputSchema.ValidateInputs(j.Inputs); err != nil {
		return nil, fmt.Errorf("inputs invalid: %w", err)
	}

	provider, err := p.factory.New(ctx, j.Provider)
	if err != nil {
		return nil, fmt.Errorf("provider: %w", err)
	}
	logger.Info("job.provider_selected", slog.String("provider", provider.Name()))

	userPrompt, err := buildUserPrompt(ctx, p.storage, j, inputSchema, outputSchema)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	outputJSONSchema, err := outputSchema.ToJSONSchemaForOutput()
	if err != nil {
		return nil, fmt.Errorf("build output json schema: %w", err)
	}

	systemPrompt := ""
	if j.SystemPrompt != nil {
		systemPrompt = *j.SystemPrompt
	}
	logger.Info("llm.request.sent",
		slog.String("provider", provider.Name()),
		slog.Int("user_prompt_chars", len(userPrompt)),
	)
	resp, err := provider.Generate(ctx, systemPrompt, userPrompt, outputJSONSchema)
	if err != nil {
		return nil, fmt.Errorf("llm call: %w", err)
	}
	logger.Info("llm.response.received", slog.Int("response_chars", len(resp)))

	parsed, validateErr := parseAndValidate(resp, outputSchema)
	if validateErr != nil && p.cfg.LLM.ValidationRetries > 0 {
		logger.Warn("llm.validation_failed", slog.Any("error", validateErr))
		retryPrompt := userPrompt +
			"\n\nThe previous response was rejected by the validator: " + validateErr.Error() +
			"\nReturn ONLY a corrected JSON object that conforms to the schema."
		resp, err = provider.Generate(ctx, systemPrompt, retryPrompt, outputJSONSchema)
		if err != nil {
			return nil, fmt.Errorf("llm retry: %w", err)
		}
		logger.Info("llm.response.received_retry")
		parsed, validateErr = parseAndValidate(resp, outputSchema)
	}
	if validateErr != nil {
		return nil, fmt.Errorf("output failed validation: %w", validateErr)
	}
	return parsed, nil
}

// parseAndValidate strips any surrounding code fence the model might emit,
// JSON-unmarshals to map[string]any, then validates against the schema.
func parseAndValidate(resp string, outputSchema schema.Schema) (json.RawMessage, error) {
	clean := stripFences(strings.TrimSpace(resp))
	if !json.Valid([]byte(clean)) {
		return nil, errors.New("response is not valid JSON")
	}
	var v any
	if err := json.Unmarshal([]byte(clean), &v); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if _, ok := v.(map[string]any); !ok {
		return nil, errors.New("response root must be a JSON object")
	}
	if err := outputSchema.ValidateInputs(json.RawMessage(clean)); err != nil {
		return nil, err
	}
	return json.RawMessage(clean), nil
}

// stripFences removes ```json ... ``` wrappers some models add despite
// JSON-mode instructions.
func stripFences(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if i := strings.IndexByte(s, '\n'); i > 0 {
		s = s[i+1:]
	}
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	return strings.TrimSpace(s)
}

// buildUserPrompt assembles the user-facing message: the prompt, then
// the inputs JSON, then any file content (base64) if present.
func buildUserPrompt(
	ctx context.Context,
	store *storage.Client,
	j sqlc.Job,
	in schema.Schema,
	out schema.Schema,
) (string, error) {
	var b strings.Builder
	b.WriteString(j.Prompt)
	b.WriteString("\n\n")
	b.WriteString(in.ToPromptHint("Inputs"))
	b.WriteString("\n")
	b.WriteString(out.ToPromptHint("Expected output"))
	b.WriteString("\nInputs JSON:\n")
	b.Write(j.Inputs)
	if j.FileBlobKey != nil && *j.FileBlobKey != "" {
		obj, err := store.GetObjectStream(ctx, *j.FileBlobKey)
		if err != nil {
			return "", fmt.Errorf("download file: %w", err)
		}
		defer obj.Close()
		raw, err := io.ReadAll(obj)
		if err != nil {
			return "", fmt.Errorf("read file: %w", err)
		}
		ct := "application/octet-stream"
		if j.FileBlobContentType != nil && *j.FileBlobContentType != "" {
			ct = *j.FileBlobContentType
		}
		b.WriteString("\n\nAttached file (")
		b.WriteString(ct)
		b.WriteString(", base64-encoded):\n")
		b.WriteString(base64.StdEncoding.EncodeToString(raw))
	}
	return b.String(), nil
}

// truncate keeps user-visible error_message in the DB short.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
