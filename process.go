package procman

import (
	"fmt"
	"runtime/debug"
)

// Process is the basic interface for any assynchronous process launched and stopped by the process manager.
type Process interface {
	// Start should block while running the processes.
	Start() error
	// Stop should signal the Start method to return. It is not required for stop to only return after start has returned.
	Stop() error
}

func processStateString(pstate int32) string {
	switch pstate {
	case ProcessStateReady:
		return "ready"
	case ProcessStateStarting:
		return "starting"
	case ProcessStateStarted:
		return "started"
	case ProcessStateStopping:
		return "stopping"
	case ProcessStateStopped:
		return "stopped"
	case ProcessStateAborted:
		return "aborted"
	default:
		return "UNKNOWN"
	}
}

const (
	ProcessStateReady int32 = iota
	ProcessStateStarting
	ProcessStateStarted
	ProcessStateStopping
	ProcessStateStopped
	ProcessStateAborted
)

type controller struct {
	process Process
	control chan int
	state   int32
}

func (controller *controller) Start() (err error) {
	defer func() {
		if data := recover(); data != nil {
			err = fmt.Errorf("process panic when starting; %+v; %s", data, debug.Stack())
		}
	}()
	return controller.process.Start()
}

func (controller *controller) Stop() (err error) {
	defer func() {
		if data := recover(); data != nil {
			err = fmt.Errorf("process panic when stopping; %+v; %s", data, debug.Stack())
		}
	}()
	return controller.process.Stop()
}
