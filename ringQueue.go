package unicorn

import (
	"fmt"
	"sync/atomic"
)

// avoid false sharing
type cacheLine [64]byte

type RingQueue struct {
	cap   uint32
	mask  uint32
	_     cacheLine
	head  uint32
	_     cacheLine
	tail  uint32
	_     cacheLine
	items []interface{}
}

func NewRingQueue(capacity uint32) *RingQueue {
	return &RingQueue{
		cap:   capacity,
		mask:  capacity - 1,
		items: make([]interface{}, capacity),
	}
}

func (r *RingQueue) Enqueue(item interface{}) error {
	for {
		head := atomic.LoadUint32(&r.head)
		tail := atomic.LoadUint32(&r.tail)
		nextTail := (tail + 1) & r.mask
		if nextTail == head {
			return fmt.Errorf("ring queue is full")
		}
		if atomic.CompareAndSwapUint32(&r.tail, tail, nextTail) {
			r.items[tail] = item
			return nil
		}
	}
}

func (r *RingQueue) Dequeue() (interface{}, error) {
	for {
		head := atomic.LoadUint32(&r.head)
		tail := atomic.LoadUint32(&r.tail)
		if head == tail {
			return nil, fmt.Errorf("ring queue is empty")
		}
		if atomic.CompareAndSwapUint32(&r.head, head, (head+1)&r.mask) {
			return r.items[head], nil
		}
	}
}

func (r *RingQueue) Cap() uint32 {
	return r.cap
}

func (r *RingQueue) Size() uint32 {
	return (atomic.LoadUint32(&r.tail) - atomic.LoadUint32(&r.head)) & r.mask
}
