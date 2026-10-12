CREATE TABLE IF NOT EXISTS elections (
                           id            BIGSERIAL PRIMARY KEY,
                           title         TEXT NOT NULL,
                           type          TEXT NOT NULL,           -- 'parliamentary' | 'presidential' | 'mixed'
                           starts_at     TIMESTAMPTZ NOT NULL,
                           ends_at       TIMESTAMPTZ NOT NULL,
                           status        TEXT NOT NULL DEFAULT 'draft' -- draft/active/finished
);