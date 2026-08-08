//go:build !windows

package main

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestWaitForShutdownSignal(t *testing.T) {
	quit := make(chan os.Signal, 1)

	done := make(chan struct{})
	go func() {
		waitForShutdownSignal(quit)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("could not find process: %v", err)
	}
	
	err = p.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("failed to send SIGTERM: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("waitForShutdownSignal did not unblock after receiving SIGTERM")
	}
}
