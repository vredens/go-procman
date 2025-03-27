package procman

import (
	"context"
	"time"
)

// WorkerOptions for defining some worker options.
type WorkerOptions struct {
	// ShutdownTimeout sets the timeout to wait for Start() to finish after Stop() is called.
	ShutdownTimeout time.Duration
}

// NewWorker creates a wrapper around a worker function which is expected to return only after ctx.Done() or an error occurs.
// WorkerOptions, if multiple are passed, will overwrite each other unless a zero value is present.
// Remember to wrap the provided context into your own cancelable contexts.
func NewWorker(main func(ctx context.Context) error, opts ...WorkerOptions) Process {
	var options = PeriodicalOptions{
		Once: true,
	}
	for _, opt := range opts {
		options.merge(PeriodicalOptions{
			ShutdownTimeout: opt.ShutdownTimeout,
		})
	}
	return NewPeriodicalJob(0, main, options)
}
