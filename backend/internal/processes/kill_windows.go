package processes

import (
	"os/exec"
	"strconv"
)

func terminateProcess(pid int64, force bool) error {
	args := []string{"/PID", strconv.FormatInt(pid, 10)}
	if force {
		args = append([]string{"/F"}, args...)
	}
	return exec.Command("taskkill", args...).Run()
}
