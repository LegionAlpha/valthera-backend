-- +goose Up

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    type TEXT NOT NULL,
    account_id UUID NOT NULL,
    destination_account_id UUID,
    -- TODO: add a foreign key once the categories table exists.
    category_id UUID,
    amount_minor BIGINT NOT NULL,
    date DATE NOT NULL,
    comment TEXT NOT NULL DEFAULT '',
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT transactions_user_fk
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE RESTRICT,

    CONSTRAINT transactions_account_fk
        FOREIGN KEY (account_id)
        REFERENCES accounts (id)
        ON DELETE RESTRICT,

    CONSTRAINT transactions_destination_account_fk
        FOREIGN KEY (destination_account_id)
        REFERENCES accounts (id)
        ON DELETE RESTRICT,

    CONSTRAINT transactions_type_check
        CHECK (type IN ('income', 'expense', 'transfer')),

    CONSTRAINT transactions_amount_range_check
        CHECK (amount_minor BETWEEN 1 AND 10000000000000),

    CONSTRAINT transactions_shape_check
        CHECK (
            (
                type = 'transfer'
                AND destination_account_id IS NOT NULL
                AND destination_account_id <> account_id
                AND category_id IS NULL
            )
            OR (
                type IN ('income', 'expense')
                AND destination_account_id IS NULL
                AND category_id IS NOT NULL
            )
        ),

    CONSTRAINT transactions_version_positive_check
        CHECK (version > 0)
);

CREATE INDEX transactions_user_date_idx
    ON transactions (user_id, date DESC, id DESC);

CREATE INDEX transactions_account_id_idx
    ON transactions (account_id);

CREATE INDEX transactions_destination_account_id_idx
    ON transactions (destination_account_id)
    WHERE destination_account_id IS NOT NULL;

-- +goose Down

DROP TABLE transactions;
