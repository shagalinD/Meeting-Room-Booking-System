package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shdmitri/booking-service/internal/db"
	"github.com/shdmitri/booking-service/internal/domain"
)

type RoomsRepository struct {
	db   *db.Queries
	pool *pgxpool.Pool
}

func NewRoomsRepository(pool *pgxpool.Pool) *RoomsRepository {
	return &RoomsRepository{
		db:   db.New(pool),
		pool: pool,
	}
}

func (rr *RoomsRepository) Create(room *domain.Room) (string, error) {
	id, err := rr.db.CreateRoom(context.Background(), db.CreateRoomParams{
		Name:         room.Name,
		Capacity:     int32(room.Capacity),
		Floor:        0,
		HasProjector: false,
		HasSound:     false,
		IsActive:     true,
	})
	return id.String(), MapError(err)
}

func (rr *RoomsRepository) GetByID(id string) (*domain.Room, error) {
	pgUUID, err := ParseUUID(id)
	if err != nil {
		return nil, MapError(err)
	}

	roomRow, err := rr.db.GetRoomByID(context.Background(), pgUUID)
	if err != nil {
		return nil, MapError(err)
	}

	return &domain.Room{
		ID:          roomRow.ID.String(),
		Name:        roomRow.Name,
		Description: "",
		Capacity:    int(roomRow.Capacity),
	}, nil
}

func (rr *RoomsRepository) List(filter domain.RoomFilter) ([]*domain.Room, error) {
	rooms, err := rr.db.ListRooms(context.Background(), db.ListRoomsParams{
		Capacity:   int32(filter.MinCapacity),
		Capacity_2: int32(filter.MaxCapacity),
		Offset:     0,
		Limit:      100,
	})
	if err != nil {
		return nil, MapError(err)
	}

	result := make([]*domain.Room, 0, len(rooms))
	for _, room := range rooms {
		result = append(result, &domain.Room{
			ID:          room.ID.String(),
			Name:        room.Name,
			Description: "",
			Capacity:    int(room.Capacity),
		})
	}

	return result, nil
}

func (rr *RoomsRepository) Update(room *domain.Room) error {
	pgUUID, err := ParseUUID(room.ID)
	if err != nil {
		return MapError(err)
	}

	err = rr.db.UpdateRoom(context.Background(), db.UpdateRoomParams{
		Name:         room.Name,
		Capacity:     int32(room.Capacity),
		Floor:        0,
		HasProjector: false,
		HasSound:     false,
		IsActive:     true,
		ID:           pgUUID,
	})
	if err != nil {
		return MapError(err)
	}

	return nil
}

func (rr *RoomsRepository) Delete(id string) error {
	pgUUID, err := ParseUUID(id)
	if err != nil {
		return MapError(err)
	}

	commandTag, err := rr.pool.Exec(context.Background(), `DELETE FROM rooms WHERE id = $1`, pgUUID)
	if err != nil {
		return MapError(err)
	}

	if commandTag.RowsAffected() == 0 {
		return MapError(pgx.ErrNoRows)
	}

	return nil
}

