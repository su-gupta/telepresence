package cliutil

import (
	"context"
	"fmt"
	"time"
)

type withoutCancel struct {
	context.Context
}

func (withoutCancel) Deadline() (deadline time.Time, ok bool) { return }
func (withoutCancel) Done() <-chan struct{}                   { return nil }
func (withoutCancel) Err() error                              { return nil }
func (c withoutCancel) String() string                        { return fmt.Sprintf("%v.WithoutCancel", c.Context) }

func EnvPairs(env map[string]string) []string {
	pairs := make([]string, len(env))
	i := 0
	for k, v := range env {
		pairs[i] = k + "=" + v
		i++
	}
	return pairs
}
