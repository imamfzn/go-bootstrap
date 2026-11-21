-- +goose Up
CREATE TABLE orders (
    id         uuid        PRIMARY KEY,
    user_id    text        NOT NULL,
    product_id text        NOT NULL,
    quantity   integer     NOT NULL CHECK (quantity > 0),
    amount     bigint      NOT NULL CHECK (amount > 0),
    note       text        NOT NULL DEFAULT '',
    status     text        NOT NULL CHECK (status IN ('pending', 'paid', 'cancelled', 'expired')),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX orders_user_id_created_at_idx ON orders (user_id, created_at DESC);

-- Serves the expire-orders job, which only ever looks at pending rows.
CREATE INDEX orders_pending_created_at_idx ON orders (created_at) WHERE status = 'pending';

-- +goose Down
DROP TABLE orders;
