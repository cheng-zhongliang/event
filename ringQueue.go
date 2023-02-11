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
		cap:   size,
		mask:  size - 1,
		items: make([]item, size),
	}
}

func (r *RingQueue) Enqueue(value interface{}) error {
	for {
		head := atomic.LoadUint32(&r.head)
		tail := atomic.LoadUint32(&r.tail)
		nextTail := (tail + 1) & r.mask
		if nextTail == head {
			return fmt.Errorf("Full")
		}
		item := &r.items[tail]
		if atomic.CompareAndSwapUint32(&r.tail, tail, nextTail) && atomic.CompareAndSwapUint32(&item.flag, F, W) {
			item.value = item
			atomic.StoreUint32(&item.flag, F)
			return nil
		}
	}
}

func (r *RingQueue) Dequeue() (interface{}, error) {
	for {
		head := atomic.LoadUint32(&r.head)
		tail := atomic.LoadUint32(&r.tail)
		if head == tail {
			return nil, fmt.Errorf("Empty")
		}
		item := &r.items[head]
		if atomic.CompareAndSwapUint32(&r.head, head, (head+1)&r.mask) && atomic.CompareAndSwapUint32(&item.flag, F, R) {
			atomic.StoreUint32(&item.flag, F)
			return item.value, nil
		}
	}
}

func (r *RingQueue) Cap() uint32 {
	return r.cap
}

func (r *RingQueue) Size() uint32 {
	return (atomic.LoadUint32(&r.tail) - atomic.LoadUint32(&r.head)) & r.mask
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
