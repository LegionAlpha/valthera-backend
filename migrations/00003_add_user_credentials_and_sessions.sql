-- +goose Up

ALTER TABLE users
    ADD COLUMN email TEXT COLLATE "C",
    ADD COLUMN password_hash TEXT,

    ADD CONSTRAINT users_credentials_pair_check
        CHECK (
            (email IS NULL) = (password_hash IS NULL)
        ),

    ADD CONSTRAINT users_email_format_check
        CHECK (
            email = lower(btrim(email))
            AND octet_length(email) BETWEEN 3 AND 254
        ),

    ADD CONSTRAINT users_password_hash_length_check
        CHECK (
            octet_length(password_hash) BETWEEN 1 AND 1024
        ),

    ADD CONSTRAINT users_email_unique
        UNIQUE (email);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    token_hash BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,

    CONSTRAINT sessions_user_fk
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,

    CONSTRAINT sessions_token_hash_unique
        UNIQUE (token_hash),

    CONSTRAINT sessions_token_hash_length_check
        CHECK (octet_length(token_hash) = 32),

    CONSTRAINT sessions_expires_after_creation_check
        CHECK (expires_at > created_at),

    CONSTRAINT sessions_last_seen_range_check
        CHECK (
            last_seen_at >= created_at
            AND last_seen_at < expires_at
        ),

    CONSTRAINT sessions_revoked_after_creation_check
        CHECK (
            revoked_at IS NULL
            OR revoked_at >= created_at
        )
);

CREATE INDEX sessions_user_id_idx
    ON sessions (user_id);

CREATE INDEX sessions_expires_at_idx
    ON sessions (expires_at);

-- +goose Down

DROP TABLE sessions;

ALTER TABLE users
    DROP CONSTRAINT users_email_unique,
    DROP CONSTRAINT users_password_hash_length_check,
    DROP CONSTRAINT users_email_format_check,
    DROP CONSTRAINT users_credentials_pair_check,
    DROP COLUMN password_hash,
    DROP COLUMN email;