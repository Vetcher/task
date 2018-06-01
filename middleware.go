package task

import (
	"context"
	"sync/atomic"
)

type WorkplaceStatus struct {
	working  uint64
	executed uint64
	next     Worker
}

func NewWorkerStatus(next Worker) *WorkplaceStatus {
	return &WorkplaceStatus{
		next: next,
	}
}

func (s *WorkplaceStatus) Work(ctx context.Context, args ...interface{}) ([]interface{}, error) {
	atomic.AddUint64(&s.working, 1)
	atomic.AddUint64(&s.executed, 1)
	defer func() {
		atomic.AddUint64(&s.working, -1)
	}()
	return s.next.Work(ctx, args...)
}

func (s *WorkplaceStatus) Busy() bool {
	return atomic.LoadUint64(&s.working) == 0
}

func (s *WorkplaceStatus) Status() (working, executed uint64) {
	return atomic.LoadUint64(&s.working), atomic.LoadUint64(&s.executed)
}
