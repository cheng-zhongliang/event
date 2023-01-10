package unicorn

type EventCallback func(ev Event)

type EventHandler = map[int]EventCallback
