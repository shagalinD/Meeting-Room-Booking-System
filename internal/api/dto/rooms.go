package dto

type (
	Room struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Capacity    int    `json:"capacity"`
	}

	CreateRoomRequest struct {
		Name        string `json:"name"`
		Capacity    int    `json:"capacity"`
		Description string `json:"description"`
	}

	CreateRoomResponse struct {
		ID string `json:"id"`
	}

	ListRoomsResponse struct {
		Rooms []Room `json:"rooms"`
	}
)