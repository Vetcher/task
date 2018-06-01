package task

import (
	"errors"
	"sync"
	"time"

	"github.com/satori/go.uuid"
)

var (
	DirectorAlreadyWorks = errors.New("director is working already")
	DirectorNotWorks     = errors.New("director is not working yet")
)

// The main figure in working process.
type Director struct {
	workplaces map[string]workplace
	options    directorOptions
	mx         sync.Mutex
	started    bool

	execWorkplacesGroup sync.WaitGroup
}

// NewDirector creates director, which observes workers.
// Director may take some options
func NewDirector(options ...DirectorOption) *Director {
	o := defaultOptions()
	for _, option := range options {
		option(&o)
	}
	return &Director{
		workplaces: make(map[string]workplace),
		options:    o,
	}
}

// With attaches Worker to Director with given parameters, worker with parameters is a Workplace.
func (d *Director) With(worker Worker, params ...WorkerParameter) *Director {
	if d.started {
		panic(DirectorAlreadyWorks)
	}
	d.mx.Lock()
	defer d.mx.Unlock()
	p := defaultWorkerParams()
	for _, param := range params {
		param(&p)
	}
	wp := workplace{
		worker: worker,
		params: p,
		syncs:  &workplaceSyncs{},
	}
	if wp.params.id != "" {
		wp.id = wp.params.id
	} else {
		wp.id = d.options.idGenerator()
	}
	d.workplaces[wp.id] = wp
	return d
}

// Begin begins workers executions.
// Panics, when call on already working director.
func (d *Director) Begin(params ...WorkerParameter) {
	if d.started {
		panic(DirectorAlreadyWorks)
	}
	d.mx.Lock()
	defer d.mx.Unlock()
	d.started = true
	for _, param := range params {
		for _, wp := range d.workplaces {
			param(&wp.params)
		}
	}
	d.execWorkplacesGroup.Add(len(d.workplaces))
	for id := range d.workplaces {
		go d.execWorkplace(id)
	}
	d.execWorkplacesGroup.Wait()
}

// Wait waits until all workplaces done their work.
// Panics on calling not working director.
func (d *Director) Wait() {
	if !d.started {
		panic(DirectorNotWorks)
	}
	for k := range d.workplaces {
		d.workplaces[k].syncs.Wait()
	}
}

// start execution of some workplace.
// begin listening on channels, tickers and loop executions.
func (d *Director) execWorkplace(id string) {
	defer d.execWorkplacesGroup.Done()
	workplace := d.workplaces[id]
	workplace.syncs.loopSync.Add(1)
	go d.execWorkerLoop(&workplace)
	d.execWorkerTickers(&workplace)
	d.listenEventChannels(&workplace)
}

// begin execution loop.
// It may be executes limit or infinite times.
func (d *Director) execWorkerLoop(wp *workplace) {
	var (
		ok   bool
		args = wp.params.firstArgs
	)
	defer wp.syncs.loopSync.Done()
	timer := time.NewTimer(0)
	timer.Stop()
	defer timer.Stop()
	for i := newLoopController(wp.params.amountOfExecutions); i.check(); i.inc() {
		args, ok = d.execWorker(wp, args...)
		if !ok {
			return
		}
		if wp.params.delayFunc != nil {
			timer.Reset(wp.params.delayFunc())
			<-timer.C
		}
	}
}

func (d *Director) execWorkerTickers(wp *workplace) {
	wp.syncs.tickerSync.Add(len(wp.params.tickerDurations))
	for _, dur := range wp.params.tickerDurations {
		go d.execWorkerWithTicker(wp, dur)
	}
}

// executes infinite amount of times until worker execution not fails.
func (d *Director) execWorkerWithTicker(wp *workplace, duration time.Duration) {
	var (
		ok   bool
		args = wp.params.firstArgs
	)
	defer wp.syncs.tickerSync.Done()
	ticker := time.NewTicker(duration)
	for range ticker.C {
		args, ok = d.execWorker(wp, args...)
		if !ok {
			ticker.Stop()
			return
		}
	}
}

// Executes worker one time. After execution, checks parameters and concurrently executes next workers in list with results of the execution.
func (d *Director) execWorker(wp *workplace, args ...interface{}) ([]interface{}, bool) {
	var (
		results []interface{}
		err     error
	)
	if len(args) == 0 {
		results, err = wp.worker.Work(wp.params.rootContext)
	} else {
		results, err = wp.worker.Work(wp.params.rootContext, args...)
	}
	if err != nil {
		if wp.params.notifyOnError {
			wp.params.notifyOnErrorChan <- err
		}
		if wp.params.panicOnError {
			panic(err)
		}
		if wp.params.stopOnError {
			return results, false
		}
	}
	for _, id := range wp.params.nextWorkers {
		nextWp, ok := d.workplaces[id]
		if ok {
			go d.execWorker(&nextWp, results...)
		}
	}
	return results, true
}

func (d *Director) listenEventChannels(wp *workplace) {
	wp.syncs.eventSync.Add(len(wp.params.eventChannels))
	for index := range wp.params.eventChannels {
		go d.listenChannel(wp, index)
	}
}

// Begin listening on one channel and trigger execution when something comes in.
func (d *Director) listenChannel(wp *workplace, chanIndex int) {
	defer wp.syncs.eventSync.Done()
	for event := range wp.params.eventChannels[chanIndex] {
		_, ok := d.execWorker(wp, event)
		if !ok {
			return
		}
	}
}

type loopController struct {
	n   uint
	max uint
	inf bool
}

func newLoopController(max int) *loopController {
	return &loopController{
		max: uint(max),
		n:   0,
		inf: max < 0,
	}
}

func (c *loopController) check() bool {
	return c.inf || c.n < c.max
}

func (c *loopController) inc() {
	if !c.inf {
		c.n++
	}
}

func uuidGen() string {
	return uuid.NewV4().String()
}

// Generic option for director.
type DirectorOption func(*directorOptions)

func defaultOptions() directorOptions {
	return directorOptions{
		idGenerator: uuidGen,
	}
}

type directorOptions struct {
	idGenerator func() string
}

// WithIdGenerator sets generator function that should generate unique ids for workers, when id for worker not set directly.
func WithIdGenerator(generator func() string) DirectorOption {
	return func(options *directorOptions) {
		options.idGenerator = generator
	}
}
