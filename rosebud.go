package procman

import (
	"os"
	"strings"

	"github.com/mitchellh/panicwrap"
	"gitlab.com/vredens/go-logger/v2"
)

// Rosebud makes sure the last words of a great being, such as your process, will be properly recorded for mankind to
// dwell uppon and conjecture on their meaning.
// Call this as first line of your main(). You can pass the log function which receives either a stack trace of the
// original panic or an error when Rosebud failed to monitor your process for some reason. If nil is passed the default
// logger is used and on error a panic occurs.
// More info you MUST read before using this: https://github.com/mitchellh/panicwrap/blob/master/panicwrap.go.
// Last but not least, go see the movie if you didn't get the reference.
func Rosebud(logFn func(stack []string, err error)) {
	exitStatus, err := panicwrap.BasicWrap(func(output string) {
		output = strings.ReplaceAll(output, "\t", "  ")
		stack := strings.Split(output, "\n")
		if logFn == nil {
			logger.WithTags("PANIC", "ERROR").WithData(logger.KV{"stack_trace": stack}).Write("ACHTUNG %s", stack[0])
		} else {
			logFn(stack, nil)
		}
		os.Exit(1)
	})
	if err != nil {
		if logFn == nil {
			panic(err)
		} else {
			logFn(nil, err)
		}
	}
	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}
}
