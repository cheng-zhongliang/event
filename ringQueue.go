package unicorn

import (
	"fmt"
	"sync/atomic"
)

// avoid false sharing
const cacheLine = 64

const (
	F = 0 // free
	R = 1 // reading
	W = 2 // writing
)

type RingQueue struct {
	cap   uint32
	mask  uint32
	_     [cacheLine - 2*4]byte
	head  uint32
	_     [cacheLine - 4]byte
	tail  uint32
	_     [cacheLine - 4]byte
	items []item
}

type item struct {
	flag  uint32
	value interface{}
	_     [cacheLine - 4 - 8]byte
}

func NewRingQueue(capacity uint32) *RingQueue {
	size := roundUpToPower2(capacity)
	return &RingQueue{
		cap:   size - 1,
		mask:  size - 1,
		items: make([]item, size),
	}
}

func (rq *RingQueue) Enqueue(value interface{}) error {
	for {
		head := atomic.LoadUint32(&rq.head)
		tail := atomic.LoadUint32(&rq.tail)
		nextTail := (tail + 1) & rq.mask
		if nextTail == head {
			return fmt.Errorf("Full")
		}
		if atomic.CompareAndSwapUint32(&rq.tail, tail, nextTail) && atomic.CompareAndSwapUint32(&rq.items[tail].flag, F, W) {
			item := &rq.items[tail]
			item.value = value
			atomic.StoreUint32(&item.flag, F)
			return nil
		}
	}
}

func (rq *RingQueue) Dequeue() (interface{}, error) {
	for {
		head := atomic.LoadUint32(&rq.head)
		tail := atomic.LoadUint32(&rq.tail)
		if head == tail {
			return nil, fmt.Errorf("Empty")
		}
		if atomic.CompareAndSwapUint32(&rq.head, head, (head+1)&rq.mask) && atomic.CompareAndSwapUint32(&rq.items[head].flag, F, R) {
			item := &rq.items[head]
			atomic.StoreUint32(&item.flag, F)
			return item.value, nil
		}
	}
}

func (rq *RingQueue) Cap() uint32 {
	return rq.cap
}

func (rq *RingQueue) Size() uint32 {
	return (atomic.LoadUint32(&rq.tail) - atomic.LoadUint32(&rq.head)) & rq.mask
}

func roundUpToPower2(v uint32) uint32 {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}
