package task

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
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

var TimeoutError = errors.New("execution timeout")

type TimeoutMiddleware struct {
	timeout time.Duration
	next    Worker
}

func NewWorkerTimeout(next Worker, timeout time.Duration) *TimeoutMiddleware {
	return &TimeoutMiddleware{
		next:    next,
		timeout: timeout,
	}
}

func (s *TimeoutMiddleware) Work(ctx context.Context, args ...interface{}) ([]interface{}, error) {
	timer := time.NewTimer(s.timeout)
	defer timer.Stop()
	result := make(chan []interface{})
	e := make(chan error)
	go func() {
		x, err := s.next.Work(ctx, args...)
		if err != nil {
			e <- err
		} else {
			result <- x
		}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, TimeoutError
	case r := <-result:
		close(result)
		close(e)
		return r, nil
	case err := <-e:
		close(result)
		close(e)
		return nil, err
	}
}
