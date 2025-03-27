package procman

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// PeriodicalOptions for tuning periodical job execution.
type PeriodicalOptions struct {
	// Idle time between executions of your job function; has no effect if period is 0 or Once is set to true.
	Idle time.Duration
	// Once, if true, will run the job only once and then stop.
	Once bool
	// ShutdownTimeout defines the max time to wait for a job to finish after Stop() is called. Defaults to 60 seconds. Must be at least 1 second.
	ShutdownTimeout time.Duration
	// Dbgf function is called for debugging purposes when certain periodical job controller's events happen.
	Dbgf func(fmt string, args ...interface{})
}

func (opts *PeriodicalOptions) merge(new PeriodicalOptions) {
	if new.Idle > 0 {
		opts.Idle = new.Idle
	}
	if new.Once {
		opts.Once = new.Once
	}
	if new.ShutdownTimeout > 0 {
		opts.ShutdownTimeout = new.ShutdownTimeout
	}
	if new.Dbgf != nil {
		opts.Dbgf = new.Dbgf
	}
}

func (opts *PeriodicalOptions) sanitize() {
	if opts.ShutdownTimeout < time.Second {
		opts.ShutdownTimeout = 60 * time.Second
	}
	if opts.Dbgf == nil {
		opts.Dbgf = func(fmt string, args ...interface{}) {}
	}
}

type periodical struct {
	state  int32
	job    func(ctx context.Context) error
	done   chan struct{}
	stop   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	period time.Duration
	opts   PeriodicalOptions
}

// NewPeriodicalJob creates a periodical runner of a "job" function which will be executed one at a time and no more than once each period.
// A cancelable context is provided to the job and if context is canceled it should stop execution as soon as possible.
// Job is allowed to shutdown without error, on error the periodical controller stops immediately.
// Job will be executed after each period elapses unless it is already running (runs only one at a time).
// Period can be 0 for just setting up continous execution.
func NewPeriodicalJob(period time.Duration, job func(ctx context.Context) error, options ...PeriodicalOptions) Process {
	c := &periodical{
		state:  ProcessStateReady,
		done:   make(chan struct{}),
		stop:   make(chan struct{}),
		job:    job,
		period: period,
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	for _, o := range options {
		c.opts.merge(o)
	}
	c.opts.sanitize()

	return c
}

func (c *periodical) Start() (err error) {
	if !atomic.CompareAndSwapInt32(&c.state, ProcessStateReady, ProcessStateStarted) {
		return fmt.Errorf("error starting periodical job [state:%s]", processStateString(atomic.LoadInt32(&c.state)))
	}
	defer close(c.done)

	var ticker *time.Ticker
	if c.period > 0 {
		ticker = time.NewTicker(c.period)
	}
	for {
		if atomic.LoadInt32(&c.state) != ProcessStateStarted {
			c.opts.Dbgf("new iteration but periodical job is already stopped")
			return nil
		}
		c.opts.Dbgf("running periodical job")
		if err := c.job(c.ctx); err != nil {
			return err
		}
		c.opts.Dbgf("periodical job finished")
		// REVIEW: this is interesting but needs some tweaks in terms of state machine.
		// if !atomic.CompareAndSwapInt32(&c.state, ProcessStateStarted, ProcessStateReady) {
		// 	return fmt.Errorf("could not change state to ready [state:%s]", processStateString(atomic.LoadInt32(&c.state)))
		// }
		if c.opts.Once {
			return nil
		}
		if c.opts.Idle > 0 {
			c.opts.Dbgf("periodical job is idleing")
			select {
			case <-c.stop:
				c.opts.Dbgf("periodical job stopped during idle period")
				return nil
			case <-time.After(c.opts.Idle):
				c.opts.Dbgf("periodical job done idleing, back to work")
			}
		}
		select {
		case <-c.stop:
			c.opts.Dbgf("periodical job stopped")
			return nil
		default:
		}
		if c.period > 0 {
			select {
			case <-c.stop:
				c.opts.Dbgf("periodical job stopped")
				return nil
			case <-ticker.C:
				c.opts.Dbgf("periodical job period expired")
			}
		}
	}
}

func (c *periodical) Stop() error {
	if atomic.CompareAndSwapInt32(&c.state, ProcessStateReady, ProcessStateStopped) {
		c.opts.Dbgf("stopping an unstarted periodical job")
		c.cancel()
		close(c.stop)
		close(c.done)
		return nil
	}
	if !atomic.CompareAndSwapInt32(&c.state, ProcessStateStarted, ProcessStateStopping) {
		return fmt.Errorf("error stopping periodical job [state:%s]", processStateString(atomic.LoadInt32(&c.state)))
	}
	c.cancel()
	close(c.stop)
	c.opts.Dbgf("waiting for start to terminate")

	select {
	case <-c.done:
		atomic.CompareAndSwapInt32(&c.state, ProcessStateStopping, ProcessStateStopped)
		return nil
	case <-time.After(c.opts.ShutdownTimeout):
		atomic.CompareAndSwapInt32(&c.state, ProcessStateStopping, ProcessStateAborted)
		return fmt.Errorf("process terminated after shutdown timeout exceeded")
	}
}
