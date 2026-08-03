package mq

import (
	"context"
	"testing"
	"time"
)

func TestKeepAliveReturnsAfterConnectionClose(t *testing.T) {
	conn := &Conn{closed: true}
	done := make(chan struct{})
	go func() {
		conn.KeepAlive(context.Background(), time.Millisecond)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("KeepAlive did not return after connection close")
	}
}

func TestKeepAliveReturnsWhenServiceContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	conn := &Conn{}
	done := make(chan struct{})
	go func() {
		conn.KeepAlive(ctx, time.Hour)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("KeepAlive did not return after service context cancellation")
	}
}
