package ipfs

import (
	"fmt"
	"os"
	"testing"
)

var tmpDir = "tmp_data"

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Error("client is nil")
	}
}

func TestAddDir(t *testing.T) {
	client := NewClient()
	hash, err := client.AddDir("./tmp_data")
	if err != nil {
		t.Error(err)
	}

	if hash == "" {
		t.Error("expected hash to not be empty")
	}
}

func TestGatewayURL(t *testing.T) {
	client := NewClient()
	url := client.GatewayURL()
	expected := "http://127.0.0.1:8080"
	if url != expected {
		t.Fatalf("expected: %s; got: %s", expected, url)
	}
}

func TestGetIpfsGatewayPort(t *testing.T) {
	port, err := getIpfsGatewayPort()
	if err != nil {
		t.Error(err)
	}

	expected := "8080"
	if port != expected {
		t.Fatalf("expected: %s; got: %s", expected, port)
	}
}

func TestNormalizeGatewayURL(t *testing.T) {
	for i, tt := range []struct {
		in  string
		out string
	}{
		{"127.0.0.1:8080", "http://127.0.0.1:8080"},
		{"http://123.123.123.123:8080", "http://123.123.123.123:8080"},
		{"", "http://ipfs.io"},
	} {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			got := NormalizeGatewayURL(tt.in)
			if got != tt.out {
				t.Errorf("want %q, got %q", tt.out, got)
			}
		})
	}
}

// last function to run so it cleans up
// the generated test files
func TestCleanup(t *testing.T) {
	cleanUp()
}

func createDataDir() {
	os.Mkdir(tmpDir, os.ModePerm)
}

func cleanUp() {
	os.Remove(tmpDir)
}

func init() {
	createDataDir()
}
