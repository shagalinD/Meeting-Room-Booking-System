package domain

type (
	Room struct {
		ID          string
		Name        string
		Description string
		Capacity    int
	}

	RoomRepository interface {
		Create(room *Room) (string, error)
		GetByID(id string) (*Room, error)
		List(filter RoomFilter) ([]*Room, error)
		Update(room *Room) error
		Delete(id string) error
	}

	RoomFilter struct {
		MinCapacity int
		MaxCapacity int
	}

	RoomService interface {
		Create(room *Room) (string, error)
		// GetByID(id string) (*Room, error)
		List(filter RoomFilter) ([]*Room, error)
		Update(room *Room) error
		Delete(id string) error
	}
)
