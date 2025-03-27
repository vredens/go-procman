package procman_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	procman "github.com/vredens/go-procman"
)

func ExampleManager() {
	var pman = procman.NewManager()

	var w1 = &MyWorker{Name: "one"}
	pman.AddProcess("one", w1)

	var w2 = &MyWorker{Name: "two"}
	pman.AddProcess("two", w2)

	//
	// Periodical Jobs
	//
	var job = func(ctx context.Context) error {
		// your batch job here
		for i := 0; i < 1000; i++ {
			// do your work here
			select {
			case <-ctx.Done():
				// our execution was aborted during an iteration
				// time for savepoint, cleanup, etc
				return nil
			default:
				continue
			}
		}
		return nil
	}
	pman.AddProcess("periodical-1", procman.NewPeriodicalJob(time.Second, job))
	pman.AddProcess("periodical-2", procman.NewPeriodicalJob(time.Second, job, procman.PeriodicalOptions{}))

	//
	// Workers
	//
	var worker = func(ctx context.Context) error {
		for {
			// do your work here
			select {
			case <-ctx.Done():
				// this could as easily be shutting down an http server or message queue consumer.
				return nil
			default:
				continue
			}
		}
	}
	pman.AddProcess("worker-1", procman.NewWorker(worker))
	pman.AddProcess("worker-2", procman.NewWorker(worker, procman.WorkerOptions{}))

	// this simulates the part where stop should be something called externally.
	// Stop is called when SIGTERM, SIGINT or SIGUSR1 are called.
	time.AfterFunc(10*time.Second, func() {
		pman.Stop()
	})

	pman.Start()
}

// MyWorker is a sample worker which implements the Start and Stop methods required by the ProcessManager for all registered Processes.
type MyWorker struct {
	ctrl  chan struct{}
	state int32
	Name  string
}

func (w *MyWorker) Start() error {
	w.ctrl = make(chan struct{})

	if !atomic.CompareAndSwapInt32(&w.state, procman.ProcessStateReady, procman.ProcessStateStarted) {
		return fmt.Errorf("worker is not in initialized state, can not start")
	}

	var ticker = time.NewTicker(time.Second)
	for {
		select {
		case <-w.ctrl:
			fmt.Println("terminating")
			ticker.Stop()
			return nil
		case <-ticker.C:
			fmt.Printf("%s ticked\n", w.Name)
		}
	}
}

func (w *MyWorker) Stop() error {
	if !atomic.CompareAndSwapInt32(&w.state, procman.ProcessStateStarted, procman.ProcessStateStopping) {
		return fmt.Errorf("can not stop worker unless it is in a started state")
	}
	close(w.ctrl)
	return nil
}
