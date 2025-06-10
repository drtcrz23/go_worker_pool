package models

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type WorkerPool struct {
	mutex       sync.Mutex
	jobs        chan Job
	Workers     map[int]*Worker
	nextWorker  int
	waitGroup   sync.WaitGroup
	ShutdownCtx context.Context
	cancelAll   context.CancelFunc
	jobTimeout  time.Duration
}

func NewWorkerPool(bufferSize int, jobTimeout time.Duration) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		jobs:        make(chan Job, bufferSize),
		Workers:     make(map[int]*Worker),
		nextWorker:  0,
		ShutdownCtx: ctx,
		cancelAll:   cancel,
		jobTimeout:  jobTimeout,
	}
}

func (pool *WorkerPool) StartWorker() int {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	id := pool.nextWorker
	pool.nextWorker++

	ctx, cancel := context.WithCancel(pool.ShutdownCtx)
	worker := &Worker{
		id:         id,
		ctx:        ctx,
		cancel:     cancel,
		jobs:       pool.jobs,
		waitGroup:  &pool.waitGroup,
		workerPool: pool,
	}
	pool.Workers[id] = worker
	pool.waitGroup.Add(1)
	go worker.run()
	log.Printf("[Pool] Запущен воркер %d (всего: %d)", id, len(pool.Workers))
	return id
}

func (pool *WorkerPool) StopWorker(id int) {
	pool.mutex.Lock()
	worker, exists := pool.Workers[id]
	if !exists {
		pool.mutex.Unlock()
		log.Printf("[Pool] Воркер %d не найден", id)
		return
	}
	delete(pool.Workers, id)
	pool.mutex.Unlock()

	worker.cancel()
	log.Printf("[Pool] Остановлен воркер %d (осталось: %d)", id, len(pool.Workers))
}

func (pool *WorkerPool) Submit(job Job) error {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	select {
	case <-pool.ShutdownCtx.Done():
		return fmt.Errorf("пул остановлен, нельзя добавить задание")
	default:
	}

	select {
	case <-pool.ShutdownCtx.Done():
		return fmt.Errorf("пул остановлен, нельзя добавить задание")
	case pool.jobs <- job:
		return nil
	}
}

func (pool *WorkerPool) Shutdown() {
	log.Println("[Pool] Инициализация завершения пула.")

	pool.mutex.Lock()
	if pool.ShutdownCtx.Err() == nil {
		pool.cancelAll()
	}
	pool.mutex.Unlock()

	pool.waitGroup.Wait()

	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	close(pool.jobs)

	pool.cancelAll()
	log.Println("[Pool] Все воркеры завершили работу.")
}
