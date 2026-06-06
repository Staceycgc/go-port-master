//go:build !windows

package processes

import (
	"os"
	"syscall"
)

func terminateProcess(pid int64, force bool) error {
	proc, err := os.FindProcess(int(pid))
	if err != nil {
		return err
	}
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	return proc.Signal(signal)
}
