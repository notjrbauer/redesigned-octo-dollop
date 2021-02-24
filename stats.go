package gateway

import (
	"container/heap"
	"sync/atomic"
	"time"
)

// NewStats returns a new stats collector
// This should be using prometheus instead.
func NewStats() *Stats {
	h := new(intHeap)
	heap.Init(h)
	return &Stats{
		p99: h,
	}
}

// Stats wraps a p99 interval, and success/error counts
type Stats struct {
	p99          *intHeap
	p95          *intHeap
	successCount count32
	errorCount   count32
}

// Record adds a duration to the p99 heap.
func (s *Stats) Record(t time.Duration) {
	heap.Push(s.p99, t)
}

// P99 returns the current p99
func (s *Stats) P99() time.Duration {
	d := s.p99.Peek().(time.Duration)
	return d
}

// P95 returns the current p95
func (s *Stats) P95() time.Duration {
	d := s.p95.Peek().(time.Duration)
	return d
}

// Count returns the total count of request events.
func (s *Stats) Count() int32 {
	return s.successCount.get() + s.errorCount.get()
}

// Success returns the count(success)
func (s *Stats) Success() int32 {
	return s.successCount.get()
}

// Error returns the count(errors)
func (s *Stats) Error() int32 {
	return s.errorCount.get()
}

func (s *Stats) IncSuccess() int32 {
	return s.successCount.inc()
}

func (s *Stats) IncError() int32 {
	return s.errorCount.inc()
}

func (h *intHeap) Peek() interface{} {
	if h.Len() > 0 {
		old := *h
		n := len(old)
		x := old[n-1]
		return x
	}
	return time.Duration(0)
}

// Min/Max Heap
type intHeap []time.Duration

func (q intHeap) Len() int           { return len(q) }
func (q intHeap) Less(i, j int) bool { return q[i] < q[j] }
func (q intHeap) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *intHeap) Push(x interface{}) {
	*q = append(*q, x.(time.Duration))
}

func (q *intHeap) Pop() interface{} {
	temp := *q
	out := temp[len(temp)-1]
	rest := temp[:len(temp)-1]
	*q = rest
	return out
}

// atomic counter
type count32 int32

func (c *count32) inc() int32 {
	return atomic.AddInt32((*int32)(c), 1)
}

func (c *count32) get() int32 {
	return atomic.LoadInt32((*int32)(c))
}

func (c *count32) set(v int32) {
	atomic.StoreInt32((*int32)(c), v)
}
