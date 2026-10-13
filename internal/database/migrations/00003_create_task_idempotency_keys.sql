-- +goose Up
CREATE TABLE task_idempotency_keys (
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    idempotency_key UUID NOT NULL,
    request_hash BYTEA NOT NULL CHECK (octet_length(request_hash) = 32),
    response_body BYTEA,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (owner_id, idempotency_key),
    CHECK (response_body IS NULL OR octet_length(response_body) > 0)
);

CREATE INDEX task_idempotency_keys_expires_at_idx ON task_idempotency_keys (expires_at);

-- +goose Down
DROP TABLE task_idempotency_keys;
