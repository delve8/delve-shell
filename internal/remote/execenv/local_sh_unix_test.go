//go:build unix

package execenv

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestLocalExecutor_RunStreaming_cancelKillsProcessGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		var x LocalExecutor
		_, _ = x.RunStreaming(ctx, "sleep 30", &out, &out)
	}()
	time.Sleep(80 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("RunStreaming did not return after cancel")
	}
}
