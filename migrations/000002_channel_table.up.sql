CREATE TABLE IF NOT EXISTS channel
(
    id      BIGSERIAL PRIMARY KEY,
    user_id BIGINT      NOT NULL,
    name    VARCHAR(32) NOT NULL,
    created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, name),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
)