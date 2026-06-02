package service

import (
	"errors"
	"sync"
	"sync/atomic"
)

type SubscriptionMaintenanceQueue struct {
	tasks    chan func()
	mu       sync.RWMutex
	stopOnce sync.Once
	stopped  atomic.Bool
	wg       sync.WaitGroup
}

func NewSubscriptionMaintenanceQueue(workerCount, queueSize int) *SubscriptionMaintenanceQueue {
	if workerCount <= 0 {
		workerCount = 1
	}
	if queueSize <= 0 {
		queueSize = 1
	}
	q := &SubscriptionMaintenanceQueue{tasks: make(chan func(), queueSize)}
	for i := 0; i < workerCount; i++ {
		q.wg.Add(1)
		go q.worker()
	}
	return q
}

func (q *SubscriptionMaintenanceQueue) TryEnqueue(task func()) error {
	if q == nil {
		return errors.New("subscription maintenance queue is nil")
	}
	if task == nil {
		return errors.New("subscription maintenance task is nil")
	}
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.stopped.Load() {
		return errors.New("subscription maintenance queue stopped")
	}
	select {
	case q.tasks <- task:
		return nil
	default:
		return errors.New("subscription maintenance queue full")
	}
}

func (q *SubscriptionMaintenanceQueue) Stop() {
	if q == nil {
		return
	}
	q.stopOnce.Do(func() {
		q.mu.Lock()
		q.stopped.Store(true)
		close(q.tasks)
		q.mu.Unlock()
		q.wg.Wait()
	})
}

func (q *SubscriptionMaintenanceQueue) worker() {
	defer q.wg.Done()
	for task := range q.tasks {
		func() {
			defer func() { _ = recover() }()
			task()
		}()
	}
}
