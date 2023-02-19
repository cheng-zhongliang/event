package unicorn

import "testing"

func TestPush(t *testing.T) {
	q := NewEQueue(1)
	ev := Event{Fd: 1, Flag: EventRead}
	err := q.Push(ev)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPop(t *testing.T) {
	q := NewEQueue(1)
	ev := Event{Fd: 1, Flag: EventRead}
	err := q.Push(ev)
	if err != nil {
		t.Fatal(err)
	}
	ev, err = q.Pop()
	if err != nil {
		t.Fatal(err)
	}
	if ev.Fd != 1 || ev.Flag != EventRead {
		t.Fatal("pop error")
	}
}

func TestPushFull(t *testing.T) {
	q := NewEQueue(1)
	ev := Event{Fd: 1, Flag: EventRead}
	err := q.Push(ev)
	if err != nil {
		t.Fatal(err)
	}
	err = q.Push(ev)
	if err != ErrQueueFull {
		t.Fatal(err)
	}
}

func TestPopEmpty(t *testing.T) {
	q := NewEQueue(1)
	_, err := q.Pop()
	if err != ErrQueueEmpty {
		t.Fatal(err)
	}
}

func BenchmarkPush(b *testing.B) {
	q := NewEQueue(b.N)
	ev := Event{Fd: 1, Flag: EventRead}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Push(ev)
	}
}

func BenchmarkPop(b *testing.B) {
	q := NewEQueue(b.N)
	ev := Event{Fd: 1, Flag: EventRead}
	for i := 0; i < b.N; i++ {
		q.Push(ev)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Pop()
	}
}

func BenchmarkPushPop(b *testing.B) {
	q := NewEQueue(b.N)
	ev := Event{Fd: 1, Flag: EventRead}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Push(ev)
		q.Pop()
	}
}

func BenchmarkPushParallel(b *testing.B) {
	q := NewEQueue(b.N)
	ev := Event{Fd: 1, Flag: EventRead}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Push(ev)
		}
	})
}

func BenchmarkPopParallel(b *testing.B) {
	q := NewEQueue(b.N)
	ev := Event{Fd: 1, Flag: EventRead}
	for i := 0; i < b.N; i++ {
		q.Push(ev)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Pop()
		}
	})
}
