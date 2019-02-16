package ipfs

import (
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
	url, err := GatewayURL()
	if err != nil {
		t.Error(err)
	}

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
