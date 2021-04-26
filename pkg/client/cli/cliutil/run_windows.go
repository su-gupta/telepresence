package cliutil

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/telepresenceio/telepresence/v2/pkg/proc"
	"golang.org/x/sys/windows"

	"github.com/telepresenceio/telepresence/v2/pkg/client/logging"
)

func RunAsRoot(ctx context.Context, exe string, args []string) error {
	if proc.IsAdmin() {
		return Start(ctx, exe, args, false, nil, nil, nil)
	}
	cwd, _ := os.Getwd()
	verbPtr, _ := windows.UTF16PtrFromString("runas")
	exePtr, _ := windows.UTF16PtrFromString(exe)
	cwdPtr, _ := windows.UTF16PtrFromString(cwd)
	var argPtr *uint16
	if len(args) > 0 {
		argsStr := logging.ShellArgsString(args)
		argPtr, _ = windows.UTF16PtrFromString(argsStr)
	}
	return windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, windows.SW_HIDE)
}

func Start(ctx context.Context, exe string, args []string, wait bool, stdin io.Reader, stdout, stderr io.Writer, env ...string) error {
	if !wait {
		// The context should not kill it if cancelled
		ctx = &withoutCancel{ctx}

		// Start in background without a terminal window using "cmd.exe
		args = append([]string{"/C", "start", "/b", exe}, args...)
		exe = "cmd.exe"
	}
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	if !wait {
		// Ensure that the processes uses a process group of its own to prevent
		// it getting affected by <ctrl-c> in the terminal
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	}

	var err error
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("%s: %v", logging.ShellString(exe, args), err)
	}
	if !wait {
		_ = cmd.Process.Release()
		return nil
	}

	// Ensure that interrupt is propagated to the child process
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		sig := <-sigCh
		if sig == nil {
			return
		}
		_ = cmd.Process.Signal(sig)
	}()
	s, err := cmd.Process.Wait()
	if err != nil {
		return fmt.Errorf("%s: %v", logging.ShellString(exe, args), err)
	}

	sigCh <- nil
	exitCode := s.ExitCode()
	if exitCode != 0 {
		return fmt.Errorf("%s %s: exited with %d", exe, strings.Join(args, " "), exitCode)
	}
	return nil
}
