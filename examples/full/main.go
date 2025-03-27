package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	procman "github.com/vredens/go-procman"
)

// MyWorker is an example of an implementation of a Process.
type MyWorker struct {
	ctrl  chan struct{}
	state int32
	Name  string
}

// Start method must be blocking and be ready to respond to soft shutdowns.
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

// Stop should send a signal which terminates the call to Start.
// Should return immediately and not wait for Start to terminate, that is handled by the ProcessManager.
func (w *MyWorker) Stop() error {
	if !atomic.CompareAndSwapInt32(&w.state, procman.ProcessStateStarted, procman.ProcessStateStopping) {
		return fmt.Errorf("can not stop worker unless it is in a started state")
	}
	close(w.ctrl)
	return nil
}

func main() {
	var w1 = &MyWorker{Name: "one"}
	var w2 = &MyWorker{Name: "two"}
	var pman = procman.NewManager()

	pman.AddProcess("one", w1)
	pman.AddProcess("two", w2)

	pman.StatusCheck()

	// A periodical job is a function which should execute periodically. Note that if the function takes longer than the
	// specified period it will being immediately unless you specify an idle time.
	var periodicalJob = procman.NewPeriodicalJob(time.Second, func(ctx context.Context) error {
		for i := 0; i < 10; i++ {
			fmt.Printf("do your work here\n")
			time.Sleep(100 * time.Millisecond)

			select {
			case <-ctx.Done():
				fmt.Printf("terminating periodical job before completing all jobs\n")
				return nil
			default:
				continue
			}
		}
		return nil
	})
	pman.AddProcess("periodical-1", periodicalJob)

	var worker = procman.NewWorker(func(ctx context.Context) error {
		for {
			fmt.Printf("do your work here\n")
			time.Sleep(100 * time.Millisecond)

			select {
			case <-ctx.Done():
				// this could as easily be shutting down an http server or message queue consumer.
				fmt.Printf("terminating worker  completing all jobs\n")
				return nil
			default:
				continue
			}
		}
	}, procman.WorkerOptions{})
	pman.AddProcess("worker-1", worker)

	// this simulates the part where stop should be something called externally.
	// Stop is called when SIGTERM, SIGINT or SIGUSR1 are called.
	time.AfterFunc(10*time.Second, func() {
		pman.Stop()
	})

	pman.Start()
}
