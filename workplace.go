package task

import "sync"

type workplace struct {
	id     string
	worker Worker
	params workerParams
	syncs  *workplaceSyncs
}

type workplaceSyncs struct {
	loopSync   sync.WaitGroup
	tickerSync sync.WaitGroup
	eventSync  sync.WaitGroup
}

func (s *workplaceSyncs) Wait() {
	s.loopSync.Wait()
	s.tickerSync.Wait()
	s.eventSync.Wait()
}
