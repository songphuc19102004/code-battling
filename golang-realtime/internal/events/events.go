package events

import "time"

type EventType string

const (
	CORRECT_SOLUTION_SUBMITTED EventType = "CORRECT_SOLUTION_SUBMITTED"
	SOLUTION_SUBMITTED         EventType = "SOLUTION_SUBMITTED"
	PLAYER_JOINED              EventType = "PLAYER_JOINED"
	PLAYER_LEFT                EventType = "PLAYER_LEFT"
	ROOM_DELETED               EventType = "ROOM_DELETED"
)

// Event wrapper for the listener
type SseEvent struct {
	EventType EventType
	Data      any
}

type SolutionSubmitted struct {
	PlayerId      int
	RoomId        int
	QuestionId    int
	Code          string
	Language      string
	SubmittedTime time.Time
}

type CorrectSolutionResult struct {
	SolutionSubmitted SolutionSubmitted
	RoomID            int
}

type LeaderboardUpdated struct {
	RoomId int
}

type PlayerJoined struct {
	PlayerID int
	RoomID   int
}

type PlayerLeft struct {
	PlayerId int
	RoomId   int
}

type RoomDeleted struct {
	RoomId int
}
