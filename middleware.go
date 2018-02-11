package task

import (
	"context"
	"sync"
)

type WorkplaceStatus struct {
	mx      sync.Mutex
	working uint
	started uint
	next    Worker
}

func NewWorkerStatus(next Worker) *WorkplaceStatus {
	return &WorkplaceStatus{next: next}
}

func (s *WorkplaceStatus) Work(ctx context.Context, args ...interface{}) ([]interface{}, error) {
	s.mx.Lock()
	s.working++
	s.started++
	s.mx.Unlock()
	defer func() {
		s.mx.Lock()
		s.working--
		s.mx.Unlock()
	}()
	return s.next.Work(ctx, args...)
}

func (s *WorkplaceStatus) Busy() bool {
	return s.working == 0
}

func (s *WorkplaceStatus) Status() (working, started uint) {
	return s.working, s.started
}
