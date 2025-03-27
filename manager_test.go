package procman

import (
	"fmt"
	"testing"
	"time"
)

func init() {
	// logger.Reconfigure(logger.ConfigMuteWriter())
}

var errDefault = fmt.Errorf("an error")

type SampleService struct {
	slowStart bool
	slowStop  bool
}

func (ss *SampleService) Start() error {
	if ss.slowStart {
		<-WaitABlinkOfAnEye()
	}
	return nil
}

func (ss *SampleService) Stop() error {
	if ss.slowStop {
		<-WaitABlinkOfAnEye()
	}
	return nil
}

type SampleBadService struct {
	wait  bool
	panic bool
}

func (bss *SampleBadService) Start() error {
	if bss.wait {
		<-WaitABlinkOfAnEye()
	}
	if bss.panic {
		panic("big estoiro")
	}

	return errDefault
}

func (bss *SampleBadService) Stop() error {
	return nil
}

func WaitABlinkOfAnEye() <-chan time.Time {
	return time.After(40 * time.Millisecond)
}

func WaitAMillisecondTimes(n time.Duration) <-chan time.Time {
	return time.After(n * time.Millisecond)
}

func TestProcessManagerLaunchWithEmptyListOfProcesses(t *testing.T) {
	terminated := make(chan bool)

	pman := NewManager()

	go func() {
		pman.Start()
		terminated <- true
	}()

	select {
	case <-WaitABlinkOfAnEye():
		t.Fail()
	case <-terminated:
		t.Log("OK")
	}

	close(terminated)
}
func TestProcessManagerStartStop(t *testing.T) {
	sampleService := &SampleService{}
	pman := NewManager()
	pman.AddProcess("sample-01", sampleService)
	terminated := make(chan bool)

	go func() {
		pman.Start()
		terminated <- true
	}()

	<-WaitABlinkOfAnEye()

	pman.Stop()

	select {
	case <-WaitABlinkOfAnEye():
		t.Fail()
	case <-terminated:
		t.Log("OK")
	}

	close(terminated)
}

func TestProcessManagerStopsAfterProcessFails(t *testing.T) {
	sbs := &SampleBadService{wait: true}
	pman := NewManager()
	pman.AddProcess("sample-01", sbs)

	terminated := make(chan bool)
	defer close(terminated)

	go func() {
		pman.Start()
		terminated <- true
	}()

	select {
	case <-WaitAMillisecondTimes(500):
		t.Fail()
	case <-terminated:
		t.Log("OK")
	}

	sc, _ := pman.StatusCheck()
	if sc != false {
		t.Error("process manager status check should return false")
	}
}

func TestProcessManagerHandlesPanicsAndStops(t *testing.T) {
	sbs := &SampleBadService{wait: true, panic: true}
	pman := NewManager()
	pman.AddProcess("sample-01", sbs)

	terminated := make(chan bool)
	defer close(terminated)

	go func() {
		pman.Start()
		terminated <- true
	}()

	select {
	case <-WaitAMillisecondTimes(500):
		t.Fail()
	case <-terminated:
		t.Log("OK")
	}

	sc, _ := pman.StatusCheck()
	if sc != false {
		t.Error("process manager status check should return false")
	}
}

func TestProcessManagerStopsAfterOneProcessFailsInMany(t *testing.T) {
	s01 := &SampleBadService{wait: true}
	s02 := &SampleService{slowStop: true}
	s03 := &SampleService{slowStop: true}

	pman := NewManager()
	pman.AddProcess("sample-01-bad", s01)
	pman.AddProcess("sample-02", s02)
	pman.AddProcess("sample-03", s03)

	terminated := make(chan bool)
	defer close(terminated)

	go func() {
		pman.Start()
		terminated <- true
	}()

	for {
		select {
		case <-WaitABlinkOfAnEye():
			t.Log("First OK")
		case <-WaitAMillisecondTimes(500):
			t.Fail()
		case <-terminated:
			t.Log("OK")

			sc, scs := pman.StatusCheck()
			if sc != false {
				t.Error("process manager status check should return false")
			}
			for k, v := range scs {
				if k == "sample-01-bad" {
					if v != ProcessStateAborted {
						t.Errorf("expected %s to have terminated with %d but got %d", k, ProcessStateAborted, v)
					}
				} else {
					if v != ProcessStateStopped {
						t.Errorf("expected %s to have terminated with %d but got %d", k, ProcessStateStopped, v)
					}
				}
			}

			return
		}
	}
}
