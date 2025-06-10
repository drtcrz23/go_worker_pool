package models

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"
)

type Worker struct {
	id         int
	ctx        context.Context
	cancel     context.CancelFunc
	jobs       <-chan Job
	waitGroup  *sync.WaitGroup
	workerPool *WorkerPool
}

func (worker *Worker) run() {
	defer worker.waitGroup.Done()
	log.Printf("[Worker %d] Старт обработки", worker.id)
	for {
		select {
		case <-worker.ctx.Done():
			log.Printf("[Worker %d] Получен сигнал остановки", worker.id)
			return

		case job, ok := <-worker.jobs:
			if !ok {
				log.Printf("[Worker %d] Канал заданий закрыт, завершаюсь", worker.id)
				return
			}

			jobCtx, cancel := context.WithTimeout(worker.ctx, worker.workerPool.jobTimeout)
			processDone := make(chan struct{})

			go func(job Job) {
				log.Printf("[Worker %d] Начинаю обработку задания %d: %s", worker.id, job.ID, job.Data)
				time.Sleep(time.Duration(rand.Intn(500)+100) * time.Millisecond)
				close(processDone)
			}(job)

			select {
			case <-processDone:
				log.Printf("[Worker %d] Завершил задание %d", worker.id, job.ID)
			case <-jobCtx.Done():
				log.Printf("[Worker %d] Таймаут при обработке задания %d", worker.id, job.ID)
			}
			cancel()
		}
	}
}
