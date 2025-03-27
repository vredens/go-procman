package procman

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPeriodical(t *testing.T) {
	var jobFactory = func(family string) func(ctx context.Context) error {
		switch family {
		case "ok":
			return func(ctx context.Context) error {
				time.Sleep(500 * time.Millisecond)
				return nil
			}
		case "2s.no-cancel":
			return func(ctx context.Context) error {
				time.Sleep(2000 * time.Millisecond)
				return nil
			}
		case "2s.with-cancel":
			return func(ctx context.Context) error {
				select {
				case <-time.After(2000 * time.Millisecond):
					return nil
				case <-ctx.Done():
					return fmt.Errorf("not finished")
				}
			}
		case "bad":
			return func(ctx context.Context) error {
				select {
				case <-time.After(200 * time.Millisecond):
					return fmt.Errorf("bad")
				case <-ctx.Done():
					return fmt.Errorf("not finished")
				}
			}
		case "panic":
			return func(ctx context.Context) error {
				select {
				case <-time.After(200 * time.Millisecond):
					panic("bad")
				case <-ctx.Done():
					return fmt.Errorf("not finished")
				}
			}
		default:
			panic("dev poo")
		}
	}

	t.Run("panic-and-stop", func(t *testing.T) {
		var c = NewPeriodicalJob(time.Second, jobFactory("panic"))
		var steps = makeSteps(2)
		go func() {
			assert.Panics(t, func() { c.Start() })
			close(steps[0])
		}()
		go func() {
			waitFor("start", steps[0], time.Second)
			assert.Nil(t, c.Stop())
			close(steps[1])
		}()
		assert.Nil(t, waitForSteps(steps, time.Second))
	})
	t.Run("error-and-stop", func(t *testing.T) {
		var c = NewPeriodicalJob(time.Second, jobFactory("bad"))
		var steps = makeSteps(2)
		go func() {
			assert.NotNil(t, c.Start())
			close(steps[0])
		}()
		go func() {
			waitFor("start", steps[0], time.Second)
			assert.Nil(t, c.Stop())
			close(steps[1])
		}()
		assert.Nil(t, waitForSteps(steps, time.Second))
	})
	t.Run("start-after-stop", func(t *testing.T) {
		var c = NewPeriodicalJob(time.Second, jobFactory("ok"))
		var steps = makeSteps(2)
		go func() {
			<-time.After(50 * time.Millisecond)
			assert.NotNil(t, c.Start())
			close(steps[0])
		}()
		go func() {
			assert.Nil(t, c.Stop())
			close(steps[1])
		}()
		assert.Nil(t, waitForSteps(steps, time.Second))
	})
	t.Run("stop-after-stop", func(t *testing.T) {
		var c = NewPeriodicalJob(time.Second, jobFactory("ok"))
		var steps = makeSteps(3)
		go func() {
			blink()
			assert.NotNil(t, c.Start())
			close(steps[0])
		}()
		go func() {
			assert.Nil(t, c.Stop())
			close(steps[1])
		}()
		go func() {
			waitFor("first stop", steps[1], time.Second)
			assert.NotNil(t, c.Stop())
			close(steps[2])
		}()
		assert.Nil(t, waitForSteps(steps, time.Second))
	})
	t.Run("stop-after-stop-no-wait", func(t *testing.T) {
		var c = NewPeriodicalJob(time.Second, jobFactory("ok"))
		var steps = makeSteps(2)
		go func() {
			blink()
			assert.NotNil(t, c.Start())
			close(steps[0])
		}()
		go func() {
			assert.Nil(t, c.Stop())
			assert.NotNil(t, c.Stop())
			close(steps[1])
		}()
		assert.Nil(t, waitForSteps(steps, time.Second))
	})
	t.Run("start-after-start", func(t *testing.T) {
		var c = NewPeriodicalJob(time.Second, jobFactory("ok"))
		var steps = makeSteps(2)
		go func() {
			assert.Nil(t, c.Start())
			assert.NotNil(t, c.Start())
			close(steps[0])
		}()
		go func() {
			blink()
			assert.Nil(t, c.Stop())
			close(steps[1])
		}()
		assert.Nil(t, waitForSteps(steps, time.Second))
	})
	t.Run("stop-slow-job/no-cancel", func(t *testing.T) {
		var c = NewPeriodicalJob(time.Second, jobFactory("2s.no-cancel")) // a 2 second job, not respecting ctx cancel
		var steps = makeSteps(2)
		go func() {
			assert.Nil(t, c.Start())
			close(steps[0])
		}()
		go func() {
			blink()
			assert.Nil(t, c.Stop())
			close(steps[1])
		}()
		assert.Nil(t, waitForSteps(steps, 3*time.Second)) // need to wait a bit more than the job time
	})
	t.Run("stop-slow-job/with-cancel", func(t *testing.T) {
		var c = NewPeriodicalJob(time.Second, jobFactory("2s.with-cancel"))
		var steps = makeSteps(2)
		go func() {
			assert.NotNil(t, c.Start())
			close(steps[0])
		}()
		go func() {
			blink()
			assert.Nil(t, c.Stop())
			close(steps[1])
		}()
		assert.Nil(t, waitForSteps(steps, time.Second))
	})
}
