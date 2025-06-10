package main

import (
	"go_worker_poll/internal/models"
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	src := rand.NewSource(1)
	r := rand.New(src)
	r.Intn(100)
	os.Exit(m.Run())
}

func TestStartAndStopWorker(t *testing.T) {
	pool := models.NewWorkerPool(5, 1*time.Second)
	id1 := pool.StartWorker()
	_ = pool.StartWorker()
	if got := len(pool.Workers); got != 2 {
		t.Errorf("ожидалось 2 воркера, получили %d", got)
	}
	pool.StopWorker(id1)
	time.Sleep(20 * time.Millisecond)
	if got := len(pool.Workers); got != 1 {
		t.Errorf("ожидалось 1 воркер после остановки, получили %d", got)
	}
	pool.Shutdown()
}

func TestSubmitAndShutdown(t *testing.T) {
	pool := models.NewWorkerPool(2, 1*time.Second)
	pool.StartWorker()
	dones := make(chan struct{})
	go func() {
		for i := 0; i < 3; i++ {
			err := pool.Submit(models.Job{ID: i, Data: "test"})
			if err != nil {
				t.Errorf("неожиданная ошибка при Submit %d: %v", i, err)
			}
		}
		close(dones)
	}()
	select {
	case <-dones:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Submit слишком долго")
	}
	pool.Shutdown()
	if err := pool.Submit(models.Job{ID: 42, Data: "after"}); err == nil {
		t.Error("ожидали ошибку при Submit после Shutdown, получили nil")
	}
}

func TestJobTimeout(t *testing.T) {
	pool := models.NewWorkerPool(0, 50*time.Millisecond)
	pool.StartWorker()
	err := pool.Submit(models.Job{ID: 99, Data: "timeout-test"})
	if err != nil {
		t.Fatalf("не удалось Submit: %v", err)
	}
	start := time.Now()
	pool.Shutdown()
	dur := time.Since(start)
	nominal := 100 * time.Millisecond
	if dur > nominal {
		t.Errorf("Shutdown занял слишком много времени (%.0fms), ожидали менее %.0fms", dur.Seconds()*1000, nominal.Seconds()*1000)
	}
}

func TestStopNonexistentWorker(t *testing.T) {
	pool := models.NewWorkerPool(1, 500*time.Millisecond)
	pool.StopWorker(12345)
	pool.Shutdown()
}
