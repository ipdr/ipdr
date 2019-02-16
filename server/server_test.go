package server

import (
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	var err error
	go func() {
		err = Run()
	}()

	time.Sleep(1 * time.Second)
	if err != nil {
		t.Error(err)
	}

	Close()
}
