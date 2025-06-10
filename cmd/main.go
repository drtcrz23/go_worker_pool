package main

import (
	"fmt"
	"go_worker_poll/internal/models"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	pool := models.NewWorkerPool(10, 1*time.Second)

	for i := 0; i < 3; i++ {
		pool.StartWorker()
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		time.Sleep(3 * time.Second)
		log.Println("[Manager] Добавляем 2 новых воркера")
		pool.StartWorker()
		pool.StartWorker()

		time.Sleep(5 * time.Second)
		log.Println("[Manager] Останавливаем воркера c id = 1")
		pool.StopWorker(1)
	}()

	go func() {
		time.Sleep(10 * time.Second)
		log.Println("[Manager] Останавливаем первые 3 воркера")
		pool.StopWorker(0)
		pool.StopWorker(1) // повторно
		pool.StopWorker(2)
	}()

	go func() {
		jobID := 1
		for {
			select {
			case <-pool.ShutdownCtx.Done():
				return
			default:
				data := fmt.Sprintf("job-%d", jobID)
				job := models.Job{ID: jobID, Data: data}
				err := pool.Submit(job)
				if err != nil {
					log.Printf("[Producer] Не удалось добавить задание %d: %v", jobID, err)
					return
				}
				log.Printf("[Producer] Отправил задание %d", jobID)
				jobID++
				time.Sleep(200 * time.Millisecond)
			}
		}
	}()
	<-sigs
	log.Println("[Main] Получен сигнал остановки. Завершаем пул.")
	pool.Shutdown()
}
