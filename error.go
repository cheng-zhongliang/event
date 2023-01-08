package unicorn

import "errors"

var (
	ErrEventExists              = errors.New("event already exists")
	ErrEventNotExists           = errors.New("event does not exist")
	ErrInvalidDemultiplexerType = errors.New("invalid demultiplexer type")
)
