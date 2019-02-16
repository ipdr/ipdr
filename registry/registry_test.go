package registry

import (
	"fmt"
	"os"
	"testing"

	docker "github.com/miguelmota/ipdr/docker"
)

var (
	testImage    = "docker.io/miguelmota/hello-world"
	testImageTar = "hello-world.tar"
)

func init() {
	createTestTar()
}

func createTestTar() {
	client := docker.NewClient()
	err := client.PullImage(testImage)
	if err != nil {
		panic(err)
	}

	err = client.SaveImageTar(testImage, testImageTar)
	if err != nil {
		panic(err)
	}
}

func createRegistry() *Registry {
	registry := NewRegistry(&Config{
		DockerLocalRegistryHost: "docker.localhost",
		IPFSHost:                "localhost:8080",
	})

	return registry
}

func TestNew(t *testing.T) {
	registry := createRegistry()
	if registry == nil {
		t.FailNow()
	}
}

func TestPushImage(t *testing.T) {
	t.Skip()
	registry := createRegistry()
	filepath := "hello-world.tar"
	reader, err := os.Open(filepath)
	if err != nil {
		t.Error(err)
	}
	ipfsHash, err := registry.PushImage(reader)
	if err != nil {
		t.Error(err)
	}
	if ipfsHash == "" {
		t.Error("expected hash")
	}
}

func TestPushImageByID(t *testing.T) {
	t.Skip()
	client := docker.NewClient()
	err := client.LoadImageByFilepath(testImageTar)
	if err != nil {
		t.Error(err)
	}

	registry := NewRegistry(&Config{})
	ipfsHash, err := registry.PushImageByID(testImage)
	if err != nil {
		t.Error(err)
	}
	if ipfsHash == "" {
		t.Error("expected hash")
	}
}

func TestDownloadImage(t *testing.T) {
	t.Skip()
	registry := createRegistry()
	location, err := registry.DownloadImage("QmQuKQ6nmUoFZGKJLHcnqahq2xgq3xbgVsQBG6YL5eF7kh")
	if err != nil {
		t.Error(err)
	}

	fmt.Println(location)
}

func TestPullImage(t *testing.T) {
	t.Skip()
	client := docker.NewClient()
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
