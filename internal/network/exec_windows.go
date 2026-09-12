//go:build windows

package network

import (
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

func prepareProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NEW_PROCESS_GROUP
}

func terminateProcessTree(cmd *exec.Cmd, wait time.Duration) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	_ = exec.Command("taskkill", "/T", "/PID", pid).Run()
	closePipes(cmd)
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return killProcessTree(cmd)
}

func killProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	err := exec.Command("taskkill", "/F", "/T", "/PID", pid).Run()
	closePipes(cmd)
	return err
}
