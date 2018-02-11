package task

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/satori/go.uuid"
)

var (
	DirectorAlreadyWorks = errors.New("director is working already")
	DirectorNotWorks     = errors.New("director is not working yet")
)

type Director struct {
	workplaces map[string]workplace
	options    directorOptions
	mx         sync.Mutex
	started    bool

	execWorkplacesGroup sync.WaitGroup
}

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
		id:     d.options.idGenerator(),
		worker: worker,
		params: p,
		syncs:  &workplaceSyncs{},
	}
	if wp.params.id != "" {
		wp.id = wp.params.id
	}
	d.workplaces[wp.id] = wp
	return d
}

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
func (d *Director) Wait() {
	if !d.started {
		panic(DirectorNotWorks)
	}
	for k := range d.workplaces {
		d.workplaces[k].syncs.Wait()
	}
}

func (d *Director) execWorkplace(id string) {
	defer d.execWorkplacesGroup.Done()
	workplace := d.workplaces[id]
	workplace.syncs.loopSync.Add(1)
	go d.execWorkerLoop(&workplace)
	d.execWorkerTickers(&workplace)
	d.listenEventChannels(&workplace)
}

func (d *Director) execWorkerLoop(wp *workplace) {
	var (
		ok   bool
		args = wp.params.firstArgs
	)
	defer wp.syncs.loopSync.Done()
	for i := newLoopController(wp.params.workNumber); i.Check(); i.Inc() {
		args, ok = d.execWorker(wp, args...)
		if !ok {
			return
		}
		if wp.params.timeoutFunc != nil {
			time.Sleep(wp.params.timeoutFunc())
		}
	}
}

func (d *Director) execWorkerTickers(wp *workplace) {
	wp.syncs.tickerSync.Add(len(wp.params.tickerDurations))
	for _, dur := range wp.params.tickerDurations {
		go d.execWorkerWithTicker(wp, dur)
	}
}

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

func (d *Director) execWorker(wp *workplace, args ...interface{}) ([]interface{}, bool) {
	var (
		results []interface{}
		err     error
	)
	if len(args) == 0 {
		results, err = wp.worker.Work(context.TODO())
	} else {
		results, err = wp.worker.Work(context.TODO(), args...)
	}
	if err != nil {
		if wp.params.notifyOnError {
			wp.params.notifyOnErrorChan <- err
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

func (c *loopController) Check() bool {
	return c.inf || c.n < c.max
}

func (c *loopController) Inc() {
	if !c.inf {
		c.n++
	}
}

func uuidGen() string {
	return uuid.NewV4().String()
}

type DirectorOption func(*directorOptions)

func defaultOptions() directorOptions {
	return directorOptions{
		idGenerator: uuidGen,
	}
}

type directorOptions struct {
	idGenerator func() string
}

func WithIdGenerator(generator func() string) DirectorOption {
	return func(options *directorOptions) {
		options.idGenerator = generator
	}
}
