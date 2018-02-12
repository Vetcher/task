package task

import (
	"context"
	"time"
)

// Worker is an instance that should do some work.
// It takes execution context and any amount of arguments and returns any amount of results and error.
type Worker interface {
	Work(ctx context.Context, args ...interface{}) ([]interface{}, error)
}

type WorkerParameter func(*workerParams)

type workerParams struct {
	id                 string
	firstArgs          []interface{}
	timeoutFunc        func() time.Duration
	amountOfExecutions int
	nextWorkers        []string
	notifyOnError      bool
	panicOnError       bool
	notifyOnErrorChan  chan<- error
	stopOnError        bool
	tickerDurations    []time.Duration
	eventChannels      []<-chan interface{}
	rootContext        context.Context
}

func defaultWorkerParams() workerParams {
	return workerParams{
		amountOfExecutions: 0,
		stopOnError:        true,
		rootContext:        context.Background(),
	}
}

// Id sets name for worker.
// Random UUID by default.
func Id(id string) WorkerParameter {
	return func(params *workerParams) {
		params.id = id
	}
}

// WithTimeout sets timeout for loop executions.
// 0 by default.
func WithTimeout(duration time.Duration) WorkerParameter {
	return WithTimeoutFunc(func() time.Duration { return duration })
}

// WithTimeoutFunc sets timeout rule for loop executions.
func WithTimeoutFunc(timeoutFunc func() time.Duration) WorkerParameter {
	return func(params *workerParams) {
		params.timeoutFunc = timeoutFunc
	}
}

// Repeat sets amount of executions for worker. For negative values of `n` worker will executes infinitely.
// 0 by default.
func Repeat(n int) WorkerParameter {
	return func(params *workerParams) {
		params.amountOfExecutions = n
	}
}

// Infinitely is a shortcut for Repeat(-1).
func Infinitely() WorkerParameter {
	return Repeat(-1)
}

// WithArgs sets args for first worker execution.
// nil by default.
func WithArgs(args ...interface{}) WorkerParameter {
	return func(params *workerParams) {
		params.firstArgs = args
	}
}

// Listen adds channels to subscribe list. Execute worker, when something received from any channel.
func Listen(channels ...<-chan interface{}) WorkerParameter {
	return func(params *workerParams) {
		params.eventChannels = append(params.eventChannels, channels...)
	}
}

// Every adds duration to ticker list.
func Every(duration time.Duration) WorkerParameter {
	return func(params *workerParams) {
		params.tickerDurations = append(params.tickerDurations, duration)
	}
}

// NotifyError sends error to channel provided channel if any was occurred.
func NotifyError(errCh chan<- error) WorkerParameter {
	return func(params *workerParams) {
		params.notifyOnError = errCh != nil
		params.notifyOnErrorChan = errCh
	}
}

// IgnoreWorkerErrors sets flag, that continues worker execution if error was occurred.
// False by default.
func IgnoreWorkerErrors(ignore bool) WorkerParameter {
	return func(params *workerParams) {
		params.stopOnError = !ignore
	}
}

// PanicOnError sets panic flag. if error was occurred, director will panic.
// False by default.
func PanicOnError(p bool) WorkerParameter {
	return func(params *workerParams) {
		params.panicOnError = p
	}
}

// WithContext sets worker execution context.
// context.Background by default.
func WithContext(ctx context.Context) WorkerParameter {
	return func(params *workerParams) {
		params.rootContext = ctx
	}
}

// Next adds worker names to 'next' list.
// After execution main worker director concurrently executes all workers in list with provided arguments taken from main worker results.
func Next(ids ...string) WorkerParameter {
	return func(params *workerParams) {
		params.nextWorkers = append(params.nextWorkers, ids...)
	}
}
