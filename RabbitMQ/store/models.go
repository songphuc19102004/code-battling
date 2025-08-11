package store

type Player struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Password string `json:"-"`
}

type Room struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Room is unique, it cannot be duplicated in the future

type RoomPlayer struct {
	// [PK, FK]
	RoomID   int `json:"room_id"`
	PlayerID int `json:"player_id"`
	Score    int `json:"points"`
	Place    int `json:"place"`
}

// for development stage, difficulty will be represented as an integer for 1 is easy, 2 is medium, 3 is hard
type Question struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Score       int    `json:"score"`
	Difficulty  int    `json:"difficulty"`
}
type LeaderboardEntry struct {
	PlayerName string `json:"player_name"`
	Score      int    `json:"score"`
	Place      int    `json:"place"`
}
