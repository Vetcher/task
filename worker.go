package task

import (
	"context"
	"time"
)

// Worker is an instance that should do some work.
// It takes execution context and any amount of arguments and returns any amount of results and error.
// Arguments comes from triggers or executions of previous worker in chain.
type Worker interface {
	Work(ctx context.Context, args ...interface{}) ([]interface{}, error)
}

// Generic parameter for worker.
type WorkerParameter func(*workerParams)

type workerParams struct {
	id                 string
	firstArgs          []interface{}
	delayFunc          func(time.Time) time.Duration
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

// WithDelay sets constant delay for loop executions.
// 0 by default.
func WithDelay(duration time.Duration) WorkerParameter {
	return WithDelayFunc(func(time.Time) time.Duration { return duration })
}

// WithDelayFunc sets delay rule for loop executions.
func WithDelayFunc(delayFunc func(last time.Time) time.Duration) WorkerParameter {
	return func(params *workerParams) {
		params.delayFunc = delayFunc
	}
}

// Repeat sets amount of executions for worker.
// For negative values of `n` worker will executes infinitely.
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

// TriggenOn adds channels to subscribe list. Execute worker, when something received from any channel.
func TriggenOn(channels ...<-chan interface{}) WorkerParameter {
	return func(params *workerParams) {
		params.eventChannels = append(params.eventChannels, channels...)
	}
}

// Every adds duration to ticker list.
// Each Every applied as OR, so Every(time.Second), Every(time.Second*2) will start one worker each second and two workers each two seconds.
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
//
// False by default.
func IgnoreWorkerErrors(ignore bool) WorkerParameter {
	return func(params *workerParams) {
		params.stopOnError = !ignore
	}
}

// PanicOnError sets panic flag. if error was occurred, director will panic.
//
// False by default.
func PanicOnError(p bool) WorkerParameter {
	return func(params *workerParams) {
		params.panicOnError = p
	}
}

// WithContext sets worker execution context.
// Director looks at context.Done() and when it triggers, director stops all new worker executions.
// It does not stops currently working workers, which were stated as next element in chain execution by Next() declaration. In this case, you should stop it by yourself.
//
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
