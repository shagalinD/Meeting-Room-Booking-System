-- name: CreateUser :exec
INSERT INTO users (email, password_hash, first_name, last_name, role) 
VALUES ($1, $2, $3, $4, $5);

-- name: GetUserByEmail :one
SELECT id, email, password_hash, first_name, last_name, role
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, first_name, last_name, role
FROM users
WHERE id = $1;