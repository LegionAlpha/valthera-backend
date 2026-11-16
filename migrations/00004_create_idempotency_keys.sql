-- +goose Up

CREATE TABLE idempotency_keys (
    user_id UUID NOT NULL,
    operation TEXT NOT NULL,
    key TEXT NOT NULL COLLATE "C",
    request_fingerprint BYTEA NOT NULL,
    resource_id UUID,
    response_status SMALLINT,
    response_headers JSONB,
    response_body BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT idempotency_keys_pkey
        PRIMARY KEY (user_id, operation, key),

    CONSTRAINT idempotency_keys_user_fk
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE RESTRICT,

    CONSTRAINT idempotency_keys_key_length_check
        CHECK (octet_length(key) BETWEEN 1 AND 255),

    CONSTRAINT idempotency_keys_operation_length_check
        CHECK (octet_length(operation) BETWEEN 1 AND 100),

    CONSTRAINT idempotency_keys_fingerprint_length_check
        CHECK (octet_length(request_fingerprint) = 32),

    CONSTRAINT idempotency_keys_status_check
        CHECK (response_status BETWEEN 200 AND 299),

    CONSTRAINT idempotency_keys_response_complete_check
        CHECK (
            (
                resource_id IS NULL
                AND response_status IS NULL
                AND response_headers IS NULL
                AND response_body IS NULL
            )
            OR (
                resource_id IS NOT NULL
                AND response_status IS NOT NULL
                AND response_headers IS NOT NULL
                AND response_body IS NOT NULL
            )
        )
);

-- +goose Down

DROP TABLE idempotency_keys;
