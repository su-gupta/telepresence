package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
)

const (
	// ConnectorSocketName is the path used when communicating to the connector process
	ConnectorSocketName = "/tmp/telepresence-connector.socket"

	// DaemonSocketName is the path used when communicating to the daemon process
	DaemonSocketName = "/var/run/telepresence-daemon.socket"
)

// SocketExists returns true if a socket is found at the given path
func SocketExists(path string) bool {
	s, err := os.Stat(path)
	return err == nil && s.Mode()&os.ModeSocket != 0
}

// WaitUntilSocketVanishes waits until the socket at the given path is removed
// and returns when that happens. The wait will be max ttw (time to wait) long.
// An error is returned if that time is exceeded before the socket is removed.
func WaitUntilSocketVanishes(name, path string, ttw time.Duration) (err error) {
	giveUp := time.Now().Add(ttw)
	for giveUp.After(time.Now()) {
		_, err = os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				err = nil
			}
			return err
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("timeout while waiting for %s to exit", name)
}

// WaitUntilSocketAppears waits until the socket at the given path comes into
// existence and returns when that happens. The wait will be max ttw (time to wait) long.
// An error is returned if that time is exceeded before the socket is removed.
func WaitUntilSocketAppears(name, path string, ttw time.Duration) (err error) {
	giveUp := time.Now().Add(ttw)
	for giveUp.After(time.Now()) {
		_, err = os.Stat(path)
		if err == nil {
			return
		}
		if !os.IsNotExist(err) {
			return err
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("timeout while waiting for %s to start", name)
}

// SocketURL returns the URL that corresponds to the given unix socket filesystem path.
func SocketURL(socket string) string {
	// The unix URL scheme was implemented in google.golang.org/grpc v1.34.0
	return "unix:" + socket
}

// DialSocket dials the given unix socket and returns the resulting connection
func DialSocket(ctx context.Context, socketName string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second) // FIXME(lukeshu): Make this configurable
	defer cancel()
	conn, err := grpc.DialContext(ctx, SocketURL(socketName),
		grpc.WithInsecure(),
		grpc.WithNoProxy(),
		grpc.WithBlock(),
	)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("socket %q exists but isn't responding; this means that either the process has locked up or has terminated ungracefully",
				SocketURL(socketName))
		}
		return nil, err
	}
	return conn, nil
}
