package gateway

import (
	"container/heap"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// Lookup is custom DNS lookup function.
// Used for static routing.
type Lookup func(service string) []net.SRV

// Scheduler schedules backends in a weighted fashion (priority queue).
type Scheduler struct {
	mu               sync.Mutex
	name             string
	backends         map[string]*queue
	services         map[string][]net.SRV
	RelookupInterval time.Duration
	CustomLookup     Lookup
	cli              Client
}

// NewScheduler makes a new scheduler.
// Also periodically looksup if set true.
func NewScheduler(interval time.Duration, custom Lookup, tags map[string][]string, socketPath string) *Scheduler {
	s := Scheduler{
		backends:         make(map[string]*queue),
		name:             "scheduler",
		services:         make(map[string][]net.SRV),
		RelookupInterval: interval,
		CustomLookup:     custom,
		cli:              NewClient(tags, socketPath),
	}
	go s.relookupEvery(interval)
	return &s
}

func (s *Scheduler) getQueue(service string) (q *queue, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q, ok = s.backends[service]
	return
}

func (s *Scheduler) getSRVs(service string) (backends []net.SRV, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	backends, ok = s.services[service]
	return
}

// NextBackend returns next backend in queue
func (s *Scheduler) NextBackend(service string) net.SRV {
	q, ok := s.getQueue(service)
	if !ok {
		s.lookup(service)
	}

	if q == nil || q.Len() == 0 {
		s.requeue(service)
	}

	return s.pop(service)
}

func (s *Scheduler) pop(service string) net.SRV {
	q, ok := s.backends[service]
	if ok && q.Len() > 0 {
		return heap.Pop(q).(net.SRV)
	}
	return net.SRV{}
}

func (s *Scheduler) requeue(service string) {
	records, _ := s.getSRVs(service)
	q := queue{}
	for i := range records {
		q = append(q, records[i])
	}
	ptr := &q
	heap.Init(ptr)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backends[service] = ptr
}

func (s *Scheduler) lookup(service string) error {
	var records []net.SRV
	addrs, err := s.cli.GetAddrs(service)
	if err != nil {
		return errors.Wrap(err, "GetAddrs")
	}

	records = make([]net.SRV, len(addrs))
	for i := 0; i < len(addrs); i++ {
		records[i] = addrs[i]
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.services[service] = records
	return nil
}

func (s *Scheduler) relookupEvery(d time.Duration) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			services := make([]string, len(s.services))
			i := 0
			for k := range s.services {
				services[i] = k
				i++
			}
			s.mu.Unlock()
			for _, service := range services {
				go s.lookup(service)
			}
		}
	}
}

type queue []net.SRV

func (q queue) Len() int           { return len(q) }
func (q queue) Less(i, j int) bool { return q[i].Priority < q[j].Priority }
func (q queue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *queue) Push(x interface{}) {
	*q = append(*q, x.(net.SRV))
}

func (q *queue) Pop() interface{} {
	temp := *q
	out := temp[len(temp)-1]
	rest := temp[:len(temp)-1]
	*q = rest
	return out
}
