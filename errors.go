package event

import "errors"

var (
	// ErrEventAlreadyAdded is returned when an event is already added.
	ErrEventAlreadyAdded = errors.New("event already added")
	// ErrEventNotAdded is returned when an event is not added.
	ErrEventNotAdded = errors.New("event not added")
)
