CREATE TABLE jobs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status                  TEXT NOT NULL CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED')),
    input_schema            JSONB NOT NULL,
    output_schema           JSONB NOT NULL,
    inputs                  JSONB NOT NULL,
    prompt                  TEXT NOT NULL,
    system_prompt           TEXT,
    file_blob_key           TEXT,
    file_blob_size          BIGINT,
    file_blob_content_type  TEXT,
    output                  JSONB,
    error_message           TEXT,
    -- "<provider>:<model>" actually used; e.g. "ollama:llama3:8b"
    provider                TEXT NOT NULL,
    attempt                 INTEGER NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at              TIMESTAMPTZ,
    completed_at            TIMESTAMPTZ
);

CREATE INDEX jobs_user_id_created_at_idx ON jobs (user_id, created_at DESC);
CREATE INDEX jobs_status_created_at_idx ON jobs (status, created_at);
