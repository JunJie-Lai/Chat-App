CREATE TABLE IF NOT EXISTS session_token
(
    hash    bytea PRIMARY KEY,
    user_id BIGINT                      NOT NULL,
    expiry  TIMESTAMP(0) WITH TIME ZONE NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
)