package domain

type Room struct {
	ID          string
	Name        string
	Description string
	Capacity    int
}

type RoomRepository interface {
	Create(room *Room) error
	GetByID(id string) (*Room, error)
	List(filter RoomFilter) ([]*Room, error)
	Update(room *Room) error
	Delete(id string) error
}

type RoomFilter struct {
	MinCapacity int
	MaxCapacity int
}