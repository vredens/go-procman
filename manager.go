// Package procman is a "process" controller which provides helper methods for managing long running go routines as
// well as creating wrappers for functions in order to convert them into a manageable process.
package procman

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"

	"gitlab.com/vredens/go-logger/v2"
)

// Manager handles your processes.
type Manager struct {
	processes map[string]*controller
	control   chan int
	mlog      logger.SLogger
	plog      logger.SLogger
	started   uint32
	mux       sync.RWMutex
}

// Parameters for the ProcessManager initializer.
type Parameters struct {
	// Logger for the different stages the manager and each process go through.
	// Defaults to slog.Default().
	Logger *slog.Logger
}

// NewManager instance using default parameters.
func NewManager() *Manager {
	return NewCustomManager(Parameters{})
}

// NewCustomManager instance.
func NewCustomManager(params Parameters) *Manager {
	if params.Logger == nil {
		params.Logger = slog.Default()
	}
	return &Manager{
		control:   make(chan int),
		processes: make(map[string]*controller),
		mlog:      logger.NewSLogWrapper(params.Logger).WithTags("pman", "manager"),
		plog:      logger.NewSLogWrapper(params.Logger).WithTags("pman", "process"),
	}
}

// IsStarted returns true if the ProcessManager has already started.
func (manager *Manager) IsStarted() bool {
	return atomic.LoadUint32(&manager.started) == 1
}

// AddProcess stores a proces in the list of processes controlled by the ProcessManager.
func (manager *Manager) AddProcess(name string, process Process) {
	if manager.IsStarted() {
		panic("can not add processes after start")
	}

	manager.processes[name] = &controller{
		process: process,
		control: make(chan int),
		state:   ProcessStateReady,
	}
}

func (manager *Manager) launch(name string, pController *controller) {
	atomic.StoreInt32(&pController.state, ProcessStateStarted)
	if err := pController.Start(); err != nil {
		manager.Stop()
		manager.plog.Errorf("[process:%s]: aborted (cause: %+v)", name, err)
		atomic.StoreInt32(&pController.state, ProcessStateAborted)
		pController.control <- 0
	} else {
		atomic.StoreInt32(&pController.state, ProcessStateStopped)
		pController.control <- 0
	}
}

// Start blocks until it receives a signal in its control channel or a SIGTERM,
// SIGINT or SIGUSR1, and should be the last method in your main.
func (manager *Manager) Start() error {
	if len(manager.processes) < 1 {
		return fmt.Errorf("no processes are registered")
	}
	if !atomic.CompareAndSwapUint32(&manager.started, 0, 1) {
		return fmt.Errorf("already started")
	}

	// listen for termination signals
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, signals...)

	manager.mux.Lock()
	manager.mlog.Infof("process manager: starting [nprocs:%d]", len(manager.processes))
	for name, process := range manager.processes {
		manager.plog.Infof("[process:%s]: starting", name)
		go manager.launch(name, process)
		manager.plog.Debugf("[process:%s]: started", name)
	}
	manager.mlog.Infof("process manager: started [nprocs:%d]", len(manager.processes))
	manager.mux.Unlock()

	select {
	case signal := <-termChan:
		manager.mlog.Infof("received signal: %s", signal.String())
	case <-manager.control:
		manager.mlog.Info("received stop")
	}

	if !atomic.CompareAndSwapUint32(&manager.started, 1, 0) {
		manager.mlog.Error("failed to change state to 'stopped'")
	}

	manager.mlog.Infof("process manager: stopping [nprocs:%d]", len(manager.processes))
	for name, process := range manager.processes {
		manager.plog.Infof("[process:%s]: stopping", name)
		if err := process.Stop(); err != nil {
			manager.plog.Errorf("[process:%s]: failed to stop (cause: %+v)", name, err)
		} else {
			manager.plog.Debugf("[process:%s]: waiting", name)
			<-process.control
			manager.plog.Infof("[process:%s]: stopped", name)
		}
	}
	manager.mlog.Infof("process manager: stopping [nprocs:%d]", len(manager.processes))

	return nil
}

// Stop will signal the ProcessManager to stop.
func (manager *Manager) Stop() {
	if atomic.CompareAndSwapUint32(&manager.started, 1, 0) {
		manager.control <- 1
	}
}

// Destroy removes all processes and closes all channels.
func (manager *Manager) Destroy() map[string]int32 {
	if manager.IsStarted() {
		manager.Stop()
	}
	out := map[string]int32{}
	for name, process := range manager.processes {
		out[name] = atomic.LoadInt32(&process.state)
		close(process.control)
		delete(manager.processes, name)
	}

	return out
}

// StatusCheck returns a tupple where the first value is a bool indicating if all processes are OK, second value is a map for de individual status of each process.
func (manager *Manager) StatusCheck() (bool, map[string]int32) {
	statuses := map[string]int32{}
	status := true

	manager.mux.RLock()
	defer manager.mux.RUnlock()

	for n, p := range manager.processes {
		statuses[n] = atomic.LoadInt32(&p.state)
		if atomic.LoadInt32(&p.state) == ProcessStateAborted {
			status = false
		}
	}

	return status, statuses
}
