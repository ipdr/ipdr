package ipfs

import (
	"os"
	"testing"
)

func init() {
	createDataDir()
}

func createDataDir() {
	os.Mkdir("tmp_data", os.ModePerm)
}

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
