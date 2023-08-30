package event

import "syscall"

const maxUint32 = 0xFFFFFFFF

// fdEvent is the event of a file descriptor.
type fdEvent struct {
	list   *list
	nread  uint8
	nwrite uint8
	nclose uint8
	nET    uint8
}

func newFdEvent() *fdEvent {
	return &fdEvent{list: newList()}
}

func (fdEv *fdEvent) add(ev *Event) {
	fdEv.list.pushBack(ev, &ev.fdEle)
	if ev.events&EvRead != 0 {
		fdEv.nread++
	}
	if ev.events&EvWrite != 0 {
		fdEv.nwrite++
	}
	if ev.events&EvClosed != 0 {
		fdEv.nclose++
	}
	if ev.events&EvET != 0 {
		fdEv.nET++
	}
}

func (fdEv *fdEvent) del(ev *Event) {
	fdEv.list.remove(&ev.fdEle)
	if ev.events&EvRead != 0 {
		fdEv.nread--
	}
	if ev.events&EvWrite != 0 {
		fdEv.nwrite--
	}
	if ev.events&EvClosed != 0 {
		fdEv.nclose--
	}
	if ev.events&EvET != 0 {
		fdEv.nET--
	}
}

func (fdEv *fdEvent) toEpollEvents() uint32 {
	epEvents := uint32(0)
	if fdEv.nread > 0 {
		epEvents |= syscall.EPOLLIN
	}
	if fdEv.nwrite > 0 {
		epEvents |= syscall.EPOLLOUT
	}
	if fdEv.nclose > 0 {
		epEvents |= syscall.EPOLLRDHUP
	}
	if epEvents != 0 && fdEv.nET > 0 {
		epEvents |= syscall.EPOLLET & maxUint32
	}
	return epEvents
}

func (fdEv *fdEvent) active(events uint32, cb func(ev *Event, res uint32)) {
	for ele := fdEv.list.front(); ele != nil; ele = ele.nextEle() {
		ev := ele.value.(*Event)
		if ev.events&(events&^EvET) != 0 {
			cb(ev, ev.events&events)
		}
	}
}
