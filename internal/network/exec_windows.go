//go:build windows

package network

import (
	"context"
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

func runTaskkill(force bool, pid string, timeout time.Duration) error {
	args := []string{"/T", "/PID", pid}
	if force {
		args = append([]string{"/F"}, args...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "taskkill", args...)
	return cmd.Run()
}

func terminateProcessTree(cmd *exec.Cmd, wait time.Duration) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	_ = runTaskkill(false, pid, wait)
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
	return runTaskkill(true, pid, DefaultExternalTerminateWait)
}
