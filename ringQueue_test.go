package unicorn

import (
	"testing"
)

func TestNewRingQueue(t *testing.T) {
	rq := NewRingQueue(1024)
	if rq == nil {
		t.Fatalf("NewRingQueue failed")
	}
	cap := rq.Cap()
	if cap != 1023 {
		t.Fatalf("NewRingQueue failed, cap != %d", cap)
	}
	size := rq.Size()
	if size != 0 {
		t.Fatalf("NewRingQueue failed, size != %d", size)
	}
}

func TestRingQueueEnqueue(t *testing.T) {
	rq := NewRingQueue(8)
	for i := 0; i < 7; i++ {
		if err := rq.Enqueue(i); err != nil {
			t.Fatalf("RingQueue.Enqueue failed, err: %s", err)
		}
	}
	size := rq.Size()
	if size != 7 {
		t.Fatalf("RingQueue.Enqueue failed, size != %d", size)
	}
}

func TestRingQueueDequeue(t *testing.T) {
	rq := NewRingQueue(8)
	for i := 0; i < 7; i++ {
		if err := rq.Enqueue(i); err != nil {
			t.Fatalf("RingQueue.Enqueue failed, err: %s", err)
		}
	}
	size := rq.Size()
	if size != 7 {
		t.Fatalf("RingQueue.Enqueue failed, size != %d", size)
	}
	for i := 0; i < 7; i++ {
		if v, err := rq.Dequeue(); err != nil {
			t.Fatalf("RingQueue.Dequeue failed, err: %s", err)
		} else {
			if vv, ok := v.(int); ok {
				if vv != i {
					t.Fatalf("RingQueue.Dequeue failed, v != %d", vv)
				}
			} else {
				t.Fatalf("RingQueue.Dequeue failed, v is not int")
			}
		}
	}
}
