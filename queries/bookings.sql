-- CREATE TABLE bookings (
--   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
--   room_id UUID NOT NULL,
--   user_id UUID NOT NULL,
--   title TEXT,
--   start_time TIMESTAMPTZ NOT NULL,
--   end_time TIMESTAMPTZ NOT NULL,
--   status booking_status NOT NULL,
--   version INTEGER NOT NULL,
--   created_at TIMESTAMPTZ DEFAULT NOW(),
--   updated_at TIMESTAMPTZ,
--   CONSTRAINT fk_room FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
--   CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
-- );

-- name: CreateBooking :one
INSERT INTO bookings (room_id, user_id, title, start_time, end_time, status, version, created_at)
VALUES ($1, $2, $3, $4, $5, $6, 1, NOW())
RETURNING id;


-- name: GetIntersections :one
SELECT * FROM bookings
WHERE room_id = $1
  AND STATUS IN ('pending', 'confirmed')
  AND tstzrange(start_time, end_time) && tstzrange($2, $3);

-- name: GetByUserId :many
SELECT * FROM bookings
WHERE user_id = $1;