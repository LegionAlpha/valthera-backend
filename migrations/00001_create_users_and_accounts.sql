-- +goose Up

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name TEXT NOT NULL,
    initial_balance_minor BIGINT NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT accounts_user_fk
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE RESTRICT,

    CONSTRAINT accounts_name_length_check
        CHECK (char_length(btrim(name)) BETWEEN 1 AND 80),

    CONSTRAINT accounts_initial_balance_range_check
        CHECK (
            initial_balance_minor BETWEEN
                -10000000000000 AND 10000000000000
        ),

    CONSTRAINT accounts_version_positive_check
        CHECK (version > 0)
);

CREATE INDEX accounts_user_id_idx ON accounts (user_id);

-- +goose Down

DROP TABLE accounts;
DROP TABLE users;