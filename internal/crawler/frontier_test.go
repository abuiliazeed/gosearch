package crawler

import (
	"container/heap"
	"testing"
)

func TestFrontier_PushPop(t *testing.T) {
	f := NewFrontier()
	defer f.Clear()

	urls := []*URL{
		{URL: "http://example.com/3", Priority: 3, Depth: 1},
		{URL: "http://example.com/1", Priority: 1, Depth: 1},
		{URL: "http://example.com/2", Priority: 2, Depth: 1},
	}

	for _, u := range urls {
		f.Push(u)
	}

	if f.Len() != 3 {
		t.Errorf("expected len 3, got %d", f.Len())
	}

	// Should pop in order of priority (lowest first)
	u1 := f.Pop()
	if u1.Priority != 1 {
		t.Errorf("expected priority 1, got %d", u1.Priority)
	}

	u2 := f.Pop()
	if u2.Priority != 2 {
		t.Errorf("expected priority 2, got %d", u2.Priority)
	}

	u3 := f.Pop()
	if u3.Priority != 3 {
		t.Errorf("expected priority 3, got %d", u3.Priority)
	}

	if f.Len() != 0 {
		t.Errorf("expected len 0, got %d", f.Len())
	}
}

func TestFrontier_Peek(t *testing.T) {
	f := NewFrontier()

	if f.Peek() != nil {
		t.Error("expected nil peek on empty frontier")
	}

	u := &URL{URL: "http://example.com", Priority: 1}
	f.Push(u)

	if f.Peek() != u {
		t.Error("expected peek to return pushed URL")
	}

	if f.Len() != 1 {
		t.Error("peek should not remove item")
	}
}

func TestPriorityQueue_Order(t *testing.T) {
	// Test depth precedence when priority is equal
	pq := make(priorityQueue, 0)
	heap.Init(&pq)

	u1 := &URL{URL: "high-depth", Priority: 1, Depth: 10}
	u2 := &URL{URL: "low-depth", Priority: 1, Depth: 1}

	heap.Push(&pq, u1)
	heap.Push(&pq, u2)

	first := heap.Pop(&pq).(*URL)
	if first.URL != "low-depth" {
		t.Error("expected lower depth to be popped first when priorities are equal")
	}
}
