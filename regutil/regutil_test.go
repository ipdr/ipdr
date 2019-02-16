package regutil

import (
	"fmt"
	"testing"
)

func TestDockerizeHash(t *testing.T) {
	for i, tt := range []struct {
		in  string
		out string
	}{
		{"QmaCm7VNsmM61FApFevTcJ1PxPabmfY4Tf2dFjpAdVdHLF", "ciqlarwwgn3qewftxxsapdx6aqi5yi7scootos7m5bqbjuzshoavapa"},
	} {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			got := DockerizeHash(tt.in)
			if got != tt.out {
				t.Errorf("want %q, got %q", tt.out, got)
			}
		})
	}
}

func TestIpfsifyHash(t *testing.T) {
	for i, tt := range []struct {
		in  string
		out string
	}{
		{"ciqlarwwgn3qewftxxsapdx6aqi5yi7scootos7m5bqbjuzshoavapa", "QmaCm7VNsmM61FApFevTcJ1PxPabmfY4Tf2dFjpAdVdHLF"},
	} {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			got := IpfsifyHash(tt.in)
			if got != tt.out {
				t.Errorf("want %q, got %q", tt.out, got)
			}
		})
	}
}
