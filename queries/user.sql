-- CREATE TABLE users (
--   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
--   email TEXT UNIQUE NOT NULL,
--   password_hash TEXT NOT NULL,
--   first_name TEXT NOT NULL,
--   last_name TEXT NOT NULL,
--   role user_role NOT NULL,
--   created_at TIMESTAMPTZ DEFAULT NOW(),
--   updated_at TIMESTAMPTZ
-- );

-- name: CreateUser :one
INSERT INTO users (email, password_hash, first_name, last_name, role) 
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, first_name, last_name, role
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, first_name, last_name, role
FROM users
WHERE id = $1;