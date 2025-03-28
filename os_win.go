//go:build windows
// +build windows

package procman

import (
	"os"
	"syscall"
)

var signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
