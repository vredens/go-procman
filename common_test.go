package procman

import (
	"fmt"
	"os"
	"time"

	"gitlab.com/vredens/go-logger/v2"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		logger.DebugMode(true)
	}
}

func flapWings() { // of a honeybee
	<-time.After(5 * time.Millisecond)
}

func changeGear() { // on my Lamborghini
	<-time.After(50 * time.Millisecond)
}

func blink() { // very fast
	<-time.After(100 * time.Millisecond)
}

func waitFor(name string, ch chan struct{}, timeout time.Duration) error {
	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timer expired waiting for %s", name)
	}
}

func waitForSteps(steps []chan struct{}, expire time.Duration) error {
	for i, step := range steps {
		select {
		case <-step:
			continue
		case <-time.After(expire):
			return fmt.Errorf("timer expired waiting for step %d", i)
		}
	}
	return nil
}

func makeSteps(n int) []chan struct{} {
	var steps = make([]chan struct{}, n)
	for i := 0; i < n; i++ {
		steps[i] = make(chan struct{})
	}
	return steps
}
