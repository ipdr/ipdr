package server

import (
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	var err error
	srv := NewServer(nil)
	go func() {
		err = srv.Start()
	}()

	time.Sleep(1 * time.Second)
	if err != nil {
		t.Error(err)
	}

	srv.Stop()
}
