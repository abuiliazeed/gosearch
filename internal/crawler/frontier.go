package crawler

import (
	"container/heap"
	"sync"
)

// Frontier is a priority queue for URLs to be crawled.
// URLs with lower priority values are crawled first.
type Frontier struct {
	pq priorityQueue
	mu sync.RWMutex
}

// NewFrontier creates a new URL frontier.
func NewFrontier() *Frontier {
	f := &Frontier{
		pq: make(priorityQueue, 0),
	}
	heap.Init(&f.pq)
	return f
}

// Push adds a URL to the frontier.
func (f *Frontier) Push(url *URL) {
	f.mu.Lock()
	defer f.mu.Unlock()
	heap.Push(&f.pq, url)
}

// Pop removes and returns the highest priority URL from the frontier.
// Returns nil if the frontier is empty.
func (f *Frontier) Pop() *URL {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.pq.Len() == 0 {
		return nil
	}
	return heap.Pop(&f.pq).(*URL)
}

// Peek returns the highest priority URL without removing it.
// Returns nil if the frontier is empty.
func (f *Frontier) Peek() *URL {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.pq.Len() == 0 {
		return nil
	}
	return f.pq[0]
}

// Len returns the number of URLs in the frontier.
func (f *Frontier) Len() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.pq.Len()
}

// Clear removes all URLs from the frontier.
func (f *Frontier) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pq = make(priorityQueue, 0)
}

// priorityQueue implements heap.Interface for URLs.
type priorityQueue []*URL

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// Lower priority value = higher priority
	// If priorities are equal, prefer shallower depth
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority < pq[j].Priority
	}
	return pq[i].Depth < pq[j].Depth
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue) Push(x interface{}) {
	item := x.(*URL)
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}
