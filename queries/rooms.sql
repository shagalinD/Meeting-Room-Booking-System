-- CREATE TABLE rooms (
--   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
--   name TEXT UNIQUE NOT NULL,
--   capacity INTEGER NOT NULL,
--   floor INTEGER NOT NULL,
--   has_projector BOOLEAN NOT NULL,
--   has_sound BOOLEAN NOT NULL,
--   is_active BOOLEAN NOT NULL,
--   created_at TIMESTAMPTZ DEFAULT NOW(),
--   updated_at TIMESTAMPTZ
-- );

-- name: CreateRoom :one
INSERT INTO rooms (name, capacity, floor, has_projector, has_sound, is_active)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: GetRoomByID :one
SELECT id, name, capacity, floor, has_projector, has_sound, is_active, created_at, updated_at
FROM rooms
WHERE id = $1;

-- name: UpdateRoom :exec
UPDATE rooms
SET name = $1, capacity = $2, floor = $3, has_projector = $4, has_sound = $5, is_active = $6, updated_at = NOW()
WHERE id = $7;

-- name: ListRooms :many
SELECT id, name, capacity, floor, has_projector, has_sound, is_active, created_at, updated_at
FROM rooms
WHERE capacity >= $1 AND capacity <= $2
ORDER BY name
OFFSET $3 LIMIT $4;
