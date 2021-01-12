package netutil

import (
	"fmt"
	"net"
	"strconv"
	"testing"
)

func TestGetFreePort(t *testing.T) {
	t.Parallel()
	port, err := GetFreePort()
	if err != nil {
		t.Error(err)
	}
	if port == 0 {
		t.Error("port is 0")
	}

	ip, err := LocalIP()
	if err != nil {
		t.Error(err)
	}

	// Try to listen on the port
	l, err := net.Listen("tcp", ip.String()+":"+strconv.Itoa(port))
	if err != nil {
		t.Error(err)
	}
	defer l.Close()
}

func TestLocalIP(t *testing.T) {
	t.Parallel()
	ip, err := LocalIP()
	if err != nil {
		t.Error(err)
	}

	if ip.String() == "" {
		t.Error("expected IP address")
	}

	t.Log(ip)
}

func TestExtractPort(t *testing.T) {
	for i, tt := range []struct {
		in  string
		out uint
	}{
		{"0.0.0.0:5000", 5000},
		{":5000", 5000},
		{"docker.local:5000", 5000},
		{"a123.com:5000", 5000},
		{"5000", 5000},
		{"", 0},
	} {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			got := ExtractPort(tt.in)
			if got != tt.out {
				t.Errorf("want %v, got %v", tt.out, got)
			}
		})
	}
}
