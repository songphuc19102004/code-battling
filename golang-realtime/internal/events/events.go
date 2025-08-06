package events

import "time"

// Generic constraints
type EventType interface {
	SolutionSubmitted | PlayerJoined | PlayerLeft
}

// Event wrapper
type Event[Type EventType] struct {
	Data *Type
}

type SolutionSubmitted struct {
	PlayerId      int
	RoomId        int
	QuestionId    int
	Code          string
	Language      string
	SubmittedTime time.Time
}

type PlayerJoined struct {
	PlayerId int
	RoomId   int
}

type PlayerLeft struct {
	PlayerId int
	RoomId   int
}

type RoomDeleted struct {
	RoomId int
}
