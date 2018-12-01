package deadline

import (
	"context"
	"fmt"
	"time"
)

// Worker is a single node in the work queue that processes jobs
type Worker struct {
	ID          int
	ticked      float64
	WorkerQueue chan Worker
	QuitChan    chan int
}

func newWorker(id int, workerQueue chan Worker) {
	worker := new(Worker)
	worker.ID = id
	worker.WorkerQueue = workerQueue
	worker.QuitChan = make(chan int)

	workerQueue <- *worker
}

// Start "starts" the worker by starting a goroutine, that is
// an infinite "for-select" loop.
func (w *Worker) Start(c context.Context, hotExit context.Context, tick float64, work Contract, done chan string) {
	timer := time.NewTicker(time.Duration(tick) * time.Millisecond)

	go func() {
		onDone := func() {
			// put worker back in workerqueue
			timer.Stop()
			w.ticked = 0
			w.WorkerQueue <- *w
		}

		for {
			select {
			case <-c.Done():
				onDone()
				return
			case <-w.QuitChan:
				onDone()
				return
			case <-hotExit.Done():
				onDone()
				return
			case <-timer.C:
				w.ticked += tick
				if IsUnixTimePast(work.TimeOut) {
					err := work.ExecOnTimeout()
					if err != nil {
						fmt.Println("Error: %v", err)
					}

					done <- work.ID
					w.Stop()
				}
			}
		}
	}()
}

// Stop tells the worker to stop listening for work requests.
// Note that the worker will only stop *after* it has finished its work.
func (w *Worker) Stop() {
	go func() {
		w.QuitChan <- 1
	}()
}
