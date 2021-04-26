package dpipe

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
)

func DPipe(ctx context.Context, cmd *dexec.Cmd, peer io.ReadWriteCloser) error {
	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to establish stdout pipe: %v", err)
	}
	defer cmdOut.Close()

	cmdIn, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to establish stdin pipe: %v", err)
	}
	defer cmdIn.Close()

	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start: %v", err)
	}

	closing := int32(0)
	go func() {
		<-ctx.Done()
		atomic.StoreInt32(&closing, 1)
		_ = peer.Close()
		if runtime.GOOS == "windows" {
			// This kills the process and any child processes that it has started. Very important when
			// killing sshfs-win since it starts a cygwin sshfs process that must be killed along with it
			_ = dexec.CommandContext(ctx, "taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
		} else {
			_ = cmd.Process.Signal(os.Interrupt)
		}
	}()

	go func() {
		if _, err := io.Copy(cmdIn, peer); err != nil && atomic.LoadInt32(&closing) == 0 {
			dlog.Errorf(ctx, "copy from sftp-server to connection failed: %v", err)
		}
	}()

	go func() {
		if _, err := io.Copy(peer, cmdOut); err != nil && atomic.LoadInt32(&closing) == 0 {
			dlog.Errorf(ctx, "copy from connection to sftp-server failed: %v", err)
		}
	}()
	if err = cmd.Wait(); err != nil && atomic.LoadInt32(&closing) == 0 {
		return fmt.Errorf("execution failed: %v", err)
	}
	return nil
}
