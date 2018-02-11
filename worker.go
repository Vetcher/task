package task

import (
	"context"
	"time"
)

type Worker interface {
	Work(ctx context.Context, args ...interface{}) ([]interface{}, error)
}

type WorkerParameter func(*workerParams)

type workerParams struct {
	id                string
	firstArgs         []interface{}
	timeoutFunc       func() time.Duration
	workNumber        int
	nextWorkers       []string
	notifyOnError     bool
	notifyOnErrorChan chan<- error
	stopOnError       bool
	tickerDurations   []time.Duration
	eventChannels     []<-chan interface{}
	rootContext       context.Context
}

func defaultWorkerParams() workerParams {
	return workerParams{
		workNumber:  0,
		stopOnError: true,
		rootContext: context.Background(),
	}
}

func Id(id string) WorkerParameter {
	return func(params *workerParams) {
		params.id = id
	}
}

func WithTimeout(duration time.Duration) WorkerParameter {
	return WithTimeoutFunc(func() time.Duration { return duration })
}

func WithTimeoutFunc(timeoutFunc func() time.Duration) WorkerParameter {
	return func(params *workerParams) {
		params.timeoutFunc = timeoutFunc
	}
}

func Repeat(n int) WorkerParameter {
	return func(params *workerParams) {
		params.workNumber = n
	}
}

func Infinity() WorkerParameter {
	return Repeat(-1)
}

func WithArgs(args ...interface{}) WorkerParameter {
	return func(params *workerParams) {
		params.firstArgs = args
	}
}

func Listen(channels ...<-chan interface{}) WorkerParameter {
	return func(params *workerParams) {
		params.eventChannels = append(params.eventChannels, channels...)
	}
}

func Every(duration time.Duration) WorkerParameter {
	return func(params *workerParams) {
		params.tickerDurations = append(params.tickerDurations, duration)
	}
}

func NotifyError(errCh chan<- error) WorkerParameter {
	return func(params *workerParams) {
		params.notifyOnError = errCh != nil
		params.notifyOnErrorChan = errCh
	}
}

func IgnoreWorkerErrors(ignore bool) WorkerParameter {
	return func(params *workerParams) {
		params.stopOnError = !ignore
	}
}

func WithContext(ctx context.Context) WorkerParameter {
	return func(params *workerParams) {
		params.rootContext = ctx
	}
}
