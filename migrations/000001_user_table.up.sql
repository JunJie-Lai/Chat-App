CREATE TABLE IF NOT EXISTS users
(
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(32)   NOT NULL,
    email         citext UNIQUE NOT NULL,
    password_hash bytea         NOT NULL
)