package procman

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorker(t *testing.T) {
	job := func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(1 * time.Second):
				return fmt.Errorf("poop")
			}
		}
	}
	worker := NewWorker(job, WorkerOptions{
		ShutdownTimeout: time.Second,
	})

	go func() {
		assert.Nil(t, worker.Start())
	}()
	<-time.After(10 * time.Millisecond)
	assert.NotNil(t, worker.Start())

	assert.Nil(t, worker.Stop())
	assert.NotNil(t, worker.Stop())
}
