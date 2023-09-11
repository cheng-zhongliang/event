package event

import "sync"

var pool = sync.Pool{
	New: func() any { return new(Event) },
}
