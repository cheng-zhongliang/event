package unicorn

type HandleFunc func(ev Event)

type EventHandler = map[Event]HandleFunc
