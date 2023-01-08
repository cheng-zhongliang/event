package unicorn

type HandleFunc func(ev Event)

type EventHandler = map[int]HandleFunc
