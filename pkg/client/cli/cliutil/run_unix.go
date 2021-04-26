// +build !windows

package cliutil

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/telepresenceio/telepresence/v2/pkg/proc"

	"golang.org/x/sys/unix"

	"github.com/telepresenceio/telepresence/v2/pkg/client/logging"
)

func RunAsRoot(ctx context.Context, exe string, args []string) error {
	if !proc.IsAdmin() {
		if err := exec.Command("sudo", "-n", "true").Run(); err != nil {
			fmt.Printf("Need root privileges to run %q\n", logging.ShellString(exe, args))
			if err = exec.Command("sudo", "true").Run(); err != nil {
				return err
			}
		}
		args = append([]string{"-n", "-E", exe}, args...)
		exe = "sudo"
	}
	return Start(ctx, exe, args, false, nil, nil, nil)
}

func Start(ctx context.Context, exe string, args []string, wait bool, stdin io.Reader, stdout, stderr io.Writer, env ...string) error {
	if !wait {
		// The context should not kill it if cancelled
		ctx = &withoutCancel{ctx}
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
		cmd.SysProcAttr = &unix.SysProcAttr{Setpgid: true}
	}

	var err error
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("%s: %w", logging.ShellString(exe, args), err)
	}
	if !wait {
		_ = cmd.Process.Release()
		return nil
	}

	// Ensure that SIGINT and SIGTERM are propagated to the child process
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, unix.SIGINT, unix.SIGTERM)
	go func() {
		sig := <-sigCh
		if sig == nil {
			return
		}
		_ = cmd.Process.Signal(sig)
	}()
	s, err := cmd.Process.Wait()
	if err != nil {
		return fmt.Errorf("%s: %w", logging.ShellString(exe, args), err)
	}

	sigCh <- nil
	exitCode := s.ExitCode()
	if exitCode != 0 {
		return fmt.Errorf("%s %s: exited with %d", exe, strings.Join(args, " "), exitCode)
	}
	return nil
}
