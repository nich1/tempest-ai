// Package config loads typed configuration from environment variables.
//
// LoadAPI and LoadConsumer return service-scoped subsets so each binary
// only sees the knobs it actually consumes. Shared infra (DB, Redis, MinIO)
// is loaded by both.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Env is the deployment environment marker. Drives Secure cookie flag,
// log format (JSON in prod, text in dev), and a handful of other defaults.
type Env string

const (
	EnvDev  Env = "dev"
	EnvProd Env = "prod"
)

// Postgres holds connection details. If DATABASE_URL is set it supersedes
// the parts.
type Postgres struct {
	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	DB       string `env:"POSTGRES_DB" envDefault:"tempest"`
	User     string `env:"POSTGRES_USER" envDefault:"tempest"`
	Password string `env:"POSTGRES_PASSWORD" envDefault:"changeme"`
	URL      string `env:"DATABASE_URL"`
}

// DSN returns the connection string. Prefers DATABASE_URL if set.
func (p Postgres) DSN() string {
	if p.URL != "" {
		return p.URL
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.Port, p.DB)
}

// Redis holds Asynq broker connection details.
type Redis struct {
	Addr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
}

// MinIO holds object storage details. Same code path works against any
// S3-compatible service in prod.
type MinIO struct {
	Endpoint  string `env:"MINIO_ENDPOINT" envDefault:"localhost:9000"`
	AccessKey string `env:"MINIO_ACCESS_KEY" envDefault:"minioadmin"`
	SecretKey string `env:"MINIO_SECRET_KEY" envDefault:"minioadmin"`
	Bucket    string `env:"MINIO_BUCKET" envDefault:"tempest-jobs"`
	UseSSL    bool   `env:"MINIO_USE_SSL" envDefault:"false"`
	Region    string `env:"MINIO_REGION" envDefault:"us-east-1"`
}

// Logging covers level and format knobs.
type Logging struct {
	Level  string `env:"LOG_LEVEL" envDefault:"info"`
	Format string `env:"LOG_FORMAT"` // "json" or "text"; if empty, derived from Env
}

// API holds API-server-specific knobs.
type API struct {
	Port                int           `env:"API_PORT" envDefault:"8080"`
	CookieName          string        `env:"COOKIE_NAME" envDefault:"session"`
	CookieDomain        string        `env:"COOKIE_DOMAIN"`
	SessionTTL          time.Duration `env:"SESSION_TTL" envDefault:"168h"`
	BcryptCost          int           `env:"BCRYPT_COST" envDefault:"12"`
	CORSAllowedOrigins  []string      `env:"CORS_ALLOWED_ORIGINS" envSeparator:"," envDefault:"http://localhost:3000"`
	MaxFileSizeBytes    int64         `env:"MAX_FILE_SIZE_BYTES" envDefault:"5368709120"`
	PresignedPutTTL     time.Duration `env:"PRESIGNED_PUT_TTL" envDefault:"15m"`
	PresignedGetTTL     time.Duration `env:"PRESIGNED_GET_TTL" envDefault:"1h"`
}

// Worker holds consumer-side knobs.
type Worker struct {
	Concurrency       int           `env:"WORKER_CONCURRENCY" envDefault:"10"`
	QueuePriorities   string        `env:"QUEUE_PRIORITIES" envDefault:"critical:6,default:3,bulk:1"`
	StrictPriority    bool          `env:"QUEUE_STRICT_PRIORITY" envDefault:"false"`
	RetryMax          int           `env:"ASYNQ_RETRY_MAX" envDefault:"5"`
	ShutdownTimeout   time.Duration `env:"ASYNQ_SHUTDOWN_TIMEOUT" envDefault:"30s"`
	PresignedGetTTL   time.Duration `env:"PRESIGNED_GET_TTL" envDefault:"1h"`
}

// LLM holds LLM provider configuration. Provider creds are only needed
// for the providers actually selected.
type LLM struct {
	ProviderDefault    string        `env:"LLM_PROVIDER_DEFAULT" envDefault:"ollama"`
	ModelDefault       string        `env:"LLM_MODEL_DEFAULT" envDefault:"llama3:8b"`
	Timeout            time.Duration `env:"LLM_TIMEOUT" envDefault:"120s"`
	ValidationRetries  int           `env:"LLM_VALIDATION_RETRIES" envDefault:"1"`
	OllamaBaseURL      string        `env:"OLLAMA_BASE_URL" envDefault:"http://localhost:11434"`
	OpenAIAPIKey       string        `env:"OPENAI_API_KEY"`
	AnthropicAPIKey    string        `env:"ANTHROPIC_API_KEY"`
	GoogleAPIKey       string        `env:"GOOGLE_API_KEY"`
}

// APIConfig is the full config bundle the API binary consumes.
type APIConfig struct {
	Env      Env      `env:"API_ENV" envDefault:"dev"`
	Postgres Postgres `envPrefix:""`
	Redis    Redis    `envPrefix:""`
	MinIO    MinIO    `envPrefix:""`
	Logging  Logging  `envPrefix:""`
	API      API      `envPrefix:""`
	LLM      LLM      `envPrefix:""` // for default provider validation
}

// ConsumerConfig is the full config bundle the consumers binary consumes.
type ConsumerConfig struct {
	Env      Env      `env:"API_ENV" envDefault:"dev"`
	Postgres Postgres `envPrefix:""`
	Redis    Redis    `envPrefix:""`
	MinIO    MinIO    `envPrefix:""`
	Logging  Logging  `envPrefix:""`
	Worker   Worker   `envPrefix:""`
	LLM      LLM      `envPrefix:""`
}

// LoadAPI parses environment variables into an APIConfig.
func LoadAPI() (APIConfig, error) {
	var c APIConfig
	if err := env.Parse(&c); err != nil {
		return APIConfig{}, fmt.Errorf("parse api config: %w", err)
	}
	c.normalize()
	return c, nil
}

// LoadConsumer parses environment variables into a ConsumerConfig.
func LoadConsumer() (ConsumerConfig, error) {
	var c ConsumerConfig
	if err := env.Parse(&c); err != nil {
		return ConsumerConfig{}, fmt.Errorf("parse consumer config: %w", err)
	}
	c.normalize()
	return c, nil
}

func (c *APIConfig) normalize() {
	c.Logging.Format = resolveLogFormat(c.Logging.Format, c.Env)
}

func (c *ConsumerConfig) normalize() {
	c.Logging.Format = resolveLogFormat(c.Logging.Format, c.Env)
}

func resolveLogFormat(format string, env Env) string {
	if format != "" {
		return format
	}
	if env == EnvProd {
		return "json"
	}
	return "text"
}

// IsProd reports whether we should treat the environment as production.
func (e Env) IsProd() bool {
	return strings.EqualFold(string(e), string(EnvProd))
}

// String redacts secret material so accidental %+v prints don't leak keys.
func (l LLM) String() string {
	return fmt.Sprintf("LLM{ProviderDefault:%s ModelDefault:%s OllamaBaseURL:%s OpenAIAPIKey:%s AnthropicAPIKey:%s GoogleAPIKey:%s}",
		l.ProviderDefault, l.ModelDefault, l.OllamaBaseURL,
		mask(l.OpenAIAPIKey), mask(l.AnthropicAPIKey), mask(l.GoogleAPIKey))
}

// String redacts the database password.
func (p Postgres) String() string {
	return fmt.Sprintf("Postgres{Host:%s Port:%d DB:%s User:%s Password:%s URL:%s}",
		p.Host, p.Port, p.DB, p.User, mask(p.Password), maskURL(p.URL))
}

// String redacts the MinIO secret key.
func (m MinIO) String() string {
	return fmt.Sprintf("MinIO{Endpoint:%s AccessKey:%s SecretKey:%s Bucket:%s UseSSL:%t Region:%s}",
		m.Endpoint, m.AccessKey, mask(m.SecretKey), m.Bucket, m.UseSSL, m.Region)
}

// String redacts the Redis password.
func (r Redis) String() string {
	return fmt.Sprintf("Redis{Addr:%s Password:%s DB:%d}",
		r.Addr, mask(r.Password), r.DB)
}

func mask(s string) string {
	if s == "" {
		return ""
	}
	return "***"
}

func maskURL(u string) string {
	if u == "" {
		return ""
	}
	return "***"
}
