package service

import (
	"errors"
	"testing"

	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
)

type roomRepositoryStub struct {
	createID string
	created  *domain.Room
	listed   domain.RoomFilter
	rooms    []*domain.Room
	updated  *domain.Room
	deleted  string
	err      error
	calls    int
}

func (r *roomRepositoryStub) Create(room *domain.Room) (string, error) {
	r.calls++
	r.created = room
	return r.createID, r.err
}

func (r *roomRepositoryStub) GetByID(string) (*domain.Room, error) {
	panic("unexpected GetByID call")
}

func (r *roomRepositoryStub) List(filter domain.RoomFilter) ([]*domain.Room, error) {
	r.calls++
	r.listed = filter
	return r.rooms, r.err
}

func (r *roomRepositoryStub) Update(room *domain.Room) error {
	r.calls++
	r.updated = room
	return r.err
}

func (r *roomRepositoryStub) Delete(id string) error {
	r.calls++
	r.deleted = id
	return r.err
}

func TestRoomServiceCreate(t *testing.T) {
	tests := []struct {
		name    string
		room    *domain.Room
		wantErr bool
	}{
		{name: "valid room", room: &domain.Room{Name: "Board room", Capacity: 8}},
		{name: "missing name", room: &domain.Room{Capacity: 8}, wantErr: true},
		{name: "zero capacity", room: &domain.Room{Name: "Board room"}, wantErr: true},
		{name: "negative capacity", room: &domain.Room{Name: "Board room", Capacity: -1}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &roomRepositoryStub{createID: "room-1"}
			id, err := (&RoomService{Repo: repo}).Create(tt.room)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				var appErr *apperrors.Errors
				if !errors.As(err, &appErr) || appErr.Code != apperrors.ValidationError {
					t.Fatalf("Create() error = %v, want validation error", err)
				}
				if repo.calls != 0 {
					t.Fatalf("repository called %d times for invalid input", repo.calls)
				}
				return
			}
			if id != "room-1" || repo.calls != 1 || repo.created != tt.room {
				t.Fatalf("Create() id=%q calls=%d room=%p, want room-1, 1, %p", id, repo.calls, repo.created, tt.room)
			}
		})
	}
}

func TestRoomServiceList(t *testing.T) {
	wantRooms := []*domain.Room{{ID: "room-1", Name: "Board room", Capacity: 8}}
	tests := []struct {
		name      string
		filter    domain.RoomFilter
		wantErr   bool
		wantCalls int
	}{
		{name: "valid filter", filter: domain.RoomFilter{MinCapacity: 2, MaxCapacity: 10}, wantCalls: 1},
		{name: "negative minimum", filter: domain.RoomFilter{MinCapacity: -1, MaxCapacity: 10}, wantErr: true},
		{name: "negative maximum", filter: domain.RoomFilter{MinCapacity: 0, MaxCapacity: -1}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &roomRepositoryStub{rooms: wantRooms}
			rooms, err := (&RoomService{Repo: repo}).List(tt.filter)
			if (err != nil) != tt.wantErr {
				t.Fatalf("List() error = %v, wantErr %v", err, tt.wantErr)
			}
			if repo.calls != tt.wantCalls {
				t.Fatalf("repository calls = %d, want %d", repo.calls, tt.wantCalls)
			}
			if !tt.wantErr && (repo.listed != tt.filter || len(rooms) != len(wantRooms) || rooms[0] != wantRooms[0]) {
				t.Fatalf("List() did not forward filter or repository result")
			}
		})
	}
}

func TestRoomServiceUpdateAndDelete(t *testing.T) {
	t.Run("update validates required fields", func(t *testing.T) {
		repo := &roomRepositoryStub{}
		service := &RoomService{Repo: repo}

		for _, room := range []*domain.Room{
			{Name: "Room", Capacity: 1},
			{ID: "room-1", Capacity: 1},
			{ID: "room-1", Name: "Room"},
		} {
			err := service.Update(room)
			var appErr *apperrors.Errors
			if !errors.As(err, &appErr) || appErr.Code != apperrors.ValidationError {
				t.Fatalf("Update(%+v) error = %v, want validation error", room, err)
			}
		}
		if repo.calls != 0 {
			t.Fatalf("repository called %d times for invalid updates", repo.calls)
		}
	})

	t.Run("update forwards valid room and repository error", func(t *testing.T) {
		wantErr := errors.New("update failed")
		room := &domain.Room{ID: "room-1", Name: "Room", Capacity: 1}
		repo := &roomRepositoryStub{err: wantErr}
		err := (&RoomService{Repo: repo}).Update(room)
		if !errors.Is(err, wantErr) || repo.updated != room || repo.calls != 1 {
			t.Fatalf("Update() error=%v updated=%p calls=%d", err, repo.updated, repo.calls)
		}
	})

	t.Run("delete validates ID", func(t *testing.T) {
		repo := &roomRepositoryStub{}
		err := (&RoomService{Repo: repo}).Delete("")
		var appErr *apperrors.Errors
		if !errors.As(err, &appErr) || appErr.Code != apperrors.ValidationError || repo.calls != 0 {
			t.Fatalf("Delete() error=%v calls=%d, want validation error without repository call", err, repo.calls)
		}
	})

	t.Run("delete forwards ID and repository error", func(t *testing.T) {
		wantErr := errors.New("delete failed")
		repo := &roomRepositoryStub{err: wantErr}
		err := (&RoomService{Repo: repo}).Delete("room-1")
		if !errors.Is(err, wantErr) || repo.deleted != "room-1" || repo.calls != 1 {
			t.Fatalf("Delete() error=%v id=%q calls=%d", err, repo.deleted, repo.calls)
		}
	})
}
