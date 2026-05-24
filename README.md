<p align="center">
  <img src="./assets/logo.png" alt="Tempest AI Logo" width="220" />
</p>

A small, distributed LLM schema manager built to demo enterprise-shaped
backend architecture: async processing, competing consumers, horizontal
scaling, queue-based workloads, structured logging, and clean service
boundaries.

You define an input/output schema and a prompt, optionally attach a file,
and submit a job. The API persists metadata, enqueues the work, and
returns immediately. A pool of horizontally-scaled Go consumers pulls
tasks off Redis (via Asynq), runs them through a pluggable LLM provider
(Ollama / OpenAI / Anthropic / Gemini), validates the response against
your output schema, and writes the structured result back to Postgres.

## Stack

| Layer    | Tech |
|----------|------|
| API      | Go, Gin, server-side sessions (HttpOnly cookies, bcrypt) |
| Queue    | Asynq on Redis (competing consumers, retries with backoff) |
| Workers  | Go, Asynq server, langchaingo for multi-provider LLMs |
| Storage  | Postgres (sqlc), MinIO (S3-compatible) for blobs |
| Frontend | Next.js (App Router), Tailwind, TypeScript |
| Infra    | Docker Compose (canonical + local override), `air` hot-reload |
| Docs     | OpenAPI 3.1 via swaggo, served at `/swagger/index.html` |

## Architecture (one-paragraph)

`apps/api` is a stateless HTTP server. It validates input/output schemas,
mints presigned MinIO upload URLs (so 5GB files never touch the API
server), persists job metadata in Postgres, and enqueues a small task on
Redis. The task carries only the `job_id` and the originating
`request_id` - everything else is fetched from Postgres at consume time
so the queue stays tiny and the DB is the source of truth. `apps/consumers`
is a stateless Asynq worker. Spin up `--scale consumers=N` and they
compete for work cleanly. `internal/*` is shared Go code (config,
logging, sqlc-generated queries, repositories, schema DSL, LLM adapters).

## Quickstart (local)

Requirements: Docker, Go 1.23+, Node 20+.

First-time setup (once per clone):

```bash
cp .env.example .env                                  # config for the Go services
cp apps/web/.env.local.example apps/web/.env.local    # config for the Next.js client
```

Boot the whole stack (every time after that):

```bash
make dev
```

Then:

- API:        http://localhost:8080
- Swagger UI: http://localhost:8080/swagger/index.html
- Web:        http://localhost:3000
- MinIO UI:   http://localhost:9001  (user: `minioadmin`, pass: `minioadmin`)
- Postgres:   `postgres://tempest:changeme@localhost:5432/tempest`
- Redis:      `redis://localhost:6379`

To horizontally scale consumers:

```bash
make scale-consumers N=4
```

## Quickstart (no docker)

First-time setup (skip if you already did it):

```bash
cp .env.example .env
cp apps/web/.env.local.example apps/web/.env.local
```

Then:

```bash
# Boot just the infra
docker compose up -d postgres redis minio minio-bootstrap

# Run the binaries directly (each in its own shell)
make api
make consumers
make web-dev
```

## API surface

All `application/json`. Auth is a session cookie set by `/auth/signup`
and `/auth/login`; clients should send credentials (`fetch`'s
`credentials: "include"`).

| Method | Path                       | Auth   | Description |
|--------|----------------------------|--------|-------------|
| POST   | `/auth/signup`             | none   | create user, start session |
| POST   | `/auth/login`              | none   | start session |
| POST   | `/auth/logout`             | cookie | revoke session |
| GET    | `/auth/me`                 | cookie | current user |
| POST   | `/jobs/file-upload-url`    | cookie | mint presigned PUT URL |
| POST   | `/jobs`                    | cookie | submit a job (returns 202) |
| GET    | `/jobs`                    | cookie | list user's jobs |
| GET    | `/jobs/{id}`               | cookie | fetch one job |
| GET    | `/health`                  | none   | deep health, queue depths |

The schema DSL accepted at `/jobs` is a list of fields with a `name`,
`type` (`string|integer|number|boolean|file|object|array`), `required`
flag, and (for object/array) nested `fields` / `items`. `file` is only
allowed at the top level of an input schema; output schemas reject it.

See [`docs/architecture.md`](docs/architecture.md) for sequence diagrams
and a deeper tour, and [`docs/swagger.yaml`](docs/swagger.yaml) for the
generated OpenAPI spec.

## Demo flow

1. Sign up at http://localhost:3000/signup
2. Submit the pre-filled sample job at http://localhost:3000/jobs
3. Watch the status flip PENDING -> PROCESSING -> COMPLETED
4. Tail the consumer logs to follow the same `request_id` end-to-end:
   ```
   docker compose logs -f consumers | grep <request_id>
   ```

## Project layout

```
apps/
  api/              HTTP server (Gin)
  consumers/        Asynq worker
  web/              Next.js frontend
internal/
  config/           env -> typed config
  logging/          slog setup, ctx helpers, pgx tracer
  db/               pgxpool + migrations + sqlc-generated queries
  users/  jobs/     thin sqlc wrappers
  auth/             bcrypt + opaque sessions
  storage/          MinIO client (presigned URLs, stat, stream)
  schema/           input/output DSL + validator + JSON Schema converter
  llm/              Provider interface + 4 langchaingo adapters + factory
  queue/            Asynq client/server/middleware/tasks
  handlers/         Gin HTTP handlers
  middleware/       request_id, logger, CORS, recoverer, RequireAuth
  processors/       LLMJobProcessor (the only Asynq handler)
docker/             Dockerfiles (prod + .dev), air configs, minio bootstrap
docs/               generated OpenAPI 3.1 (swag init), architecture.md
```

## Security choices

- **No JWTs**. Server-side sessions, opaque random tokens, sha256 at rest,
  HttpOnly + SameSite=Lax cookies, Secure flag in prod.
- **No user enumeration**. Login returns the same generic error whether
  the email is unknown or the password is wrong, with comparable timing.
- **No secrets in logs**. Postgres password / MinIO secret / API keys are
  redacted from `%+v` prints; `pgx` query args longer than 100 chars are
  redacted in the query tracer.
- **Presigned uploads**. The 5GB file size limit means the API must not
  proxy bytes; it mints presigned PUT URLs so MinIO/S3 receives the file
  directly.

## Observability

Every log line is structured (`log/slog`, JSON in prod, text in dev).
Standard fields: `service`, `request_id`, `job_id`, `user_id`, `task_id`.
The API generates a `request_id` per HTTP request, sticks it in the
response header, attaches it to the context-carried logger, and ships
it inside the Asynq task payload so consumer logs share the same id. A
single `grep` on `request_id` returns the full lifecycle of a request
across the API and the workers.

## Known limits

This is a portfolio-shaped demo, not a production deployment. In
particular: rate limiting, multi-tenancy, audit logging, encryption at
rest, and full LLM streaming are out of scope.

## License

MIT.
