package events

import (
	"golang-realtime/internal/crunner"
	"golang-realtime/internal/store"
	"time"
)

type EventType string

const (
	CORRECT_SOLUTION_SUBMITTED EventType = "CORRECT_SOLUTION_SUBMITTED"
	WRONG_SOLUTION_SUBMITTED   EventType = "WRONG_SOLUTION_SUBMITTED"
	SOLUTION_SUBMITTED         EventType = "SOLUTION_SUBMITTED"
	PLAYER_JOINED              EventType = "PLAYER_JOINED"
	PLAYER_LEFT                EventType = "PLAYER_LEFT"
	ROOM_DELETED               EventType = "ROOM_DELETED"
	COMPILATION_TEST           EventType = "COMPILATION_TEST"
)

// Event wrapper for the listener
type SseEvent struct {
	EventType EventType
	Data      any
}

type SolutionSubmitted struct {
	PlayerId      int32
	RoomId        int32
	QuestionId    int32
	Code          string
	Language      string
	SubmittedTime time.Time
}

type SolutionResult struct {
	SolutionSubmitted SolutionSubmitted
	Correct           bool
	RunOutput         crunner.RunOutput
}

type LeaderboardUpdated struct {
	RoomId int
}

type PlayerJoined struct {
	PlayerID int32
	RoomID   int32
}

type PlayerLeft struct {
	PlayerId int32
	RoomId   int32
}

type RoomDeleted struct {
	RoomId int32
}

type CompilationTest struct {
	Code     string
	Language store.Language
}
