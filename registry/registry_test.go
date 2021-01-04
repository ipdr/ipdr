package registry

import (
	"os"
	"testing"

	docker "github.com/miguelmota/ipdr/docker"
)

var (
	testImage    = "docker.io/miguelmota/hello-world"
	testImageTar = "hello-world.tar"
)

func TestNew(t *testing.T) {
	registry := createRegistry()
	if registry == nil {
		t.FailNow()
	}
}

func TestPushImage(t *testing.T) {
	registry := createRegistry()
	filepath := "hello-world.tar"
	reader, err := os.Open(filepath)
	if err != nil {
		t.Error(err)
	}
	ipfsHash, err := registry.PushImage(reader, "name:tag")
	if err != nil {
		t.Error(err)
	}
	if ipfsHash == "" {
		t.Error("expected hash")
	}
}

func TestPushImageByID(t *testing.T) {
	client := createClient()
	err := client.LoadImageByFilePath(testImageTar)
	if err != nil {
		t.Error(err)
	}

	registry := createRegistry()
	ipfsHash, err := registry.PushImageByID(testImage)
	if err != nil {
		t.Error(err)
	}
	if ipfsHash == "" {
		t.Error("expected hash")
	}
}

func TestDownloadImage(t *testing.T) {
	registry := createRegistry()
	ipfsHash, err := registry.PushImageByID(testImage)
	if err != nil {
		t.Error(err)
	}
	location, err := registry.DownloadImage(ipfsHash)
	if err != nil {
		t.Error(err)
	}

	if location == "" {
		t.Error("expected location")
	}
}

func TestPullImage(t *testing.T) {
	client := createClient()
	err := client.PullImage(testImage)
	if err != nil {
		t.Error(err)
	}

	registry := createRegistry()
	ipfsHash, err := registry.PushImageByID(testImage)
	if err != nil {
		t.Error(err)
	}

	_, err = registry.PullImage(ipfsHash)
	if err != nil {
		t.Error(err)
	}
}

// last function to run so it cleans up
// the generated test files
func TestCleanup(t *testing.T) {
	cleanUp()
}

func cleanUp() {
	os.Remove(testImageTar)
}

func createTestTar() {
	client := createClient()
	err := client.PullImage(testImage)
	if err != nil {
		panic(err)
	}

	err = client.SaveImageTar(testImage, testImageTar)
	if err != nil {
		panic(err)
	}
}

func createClient() *docker.Client {
	return docker.NewClient(nil)
}

func createRegistry() *Registry {
	registry := NewRegistry(&Config{
		DockerLocalRegistryHost: "docker.localhost:5000",
		IPFSHost:                "127.0.0.1:5001",
	})

	return registry
}

func init() {
	createTestTar()
}
