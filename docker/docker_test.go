package docker

import (
	"io"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

var (
	testImage    = "docker.io/miguelmota/hello-world"
	testImageTar = "hello-world.tar"
)

func init() {
	createTestTar()
}

func createTestTar() {
	client := NewClient()
	err := client.PullImage(testImage)
	if err != nil {
		panic(err)
	}

	err = client.SaveImageTar(testImage, "hello-world.tar")
	if err != nil {
		panic(err)
	}
}

func TestNew(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Error("expected instance")
	}
}

func TestListImages(t *testing.T) {
	client := NewClient()
	images, err := client.ListImages()
	if err != nil {
		t.Error(err)
	}

	for _, image := range images {
		if len(image.ID) == 0 {
			t.Error("expected image ID")
		}
		if image.Size <= 0 {
			t.Error("expected image size")
		}
	}
}

func TestHasImage(t *testing.T) {
	client := NewClient()
	err := client.PullImage(testImage)
	if err != nil {
		t.Error(err)
	}
	hasImage, err := client.HasImage(testImage)
	if err != nil {
		t.Error(err)
	}
	if !hasImage {
		t.Error("expected to have image")
	}
}

func TestPullImage(t *testing.T) {
	client := NewClient()
	err := client.PullImage(testImage)
	if err != nil {
		t.Error(err)
	}
}

func TestReadImage(t *testing.T) {
	client := NewClient()
	err := client.PullImage(testImage)
	if err != nil {
		t.Error(err)
	}
	reader, err := client.ReadImage(testImage)
	if err != nil {
		t.Error(err)
	}

	io.Copy(os.Stdout, reader)
}

func TestLoadImage(t *testing.T) {
	client := NewClient()
	input, err := os.Open(testImageTar)
	if err != nil {
		t.Error(err)
	}
	err = client.LoadImage(input)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadImageByFilePath(t *testing.T) {
	client := NewClient()
	err := client.LoadImageByFilePath(testImageTar)
	if err != nil {
		t.Error(err)
	}
}

func TestTagImage(t *testing.T) {
	client := NewClient()
	err := client.PullImage(testImage)
	if err != nil {
		t.Error(err)
	}

	canonicalNewTag := "docker.io/miguelmota/hello-mars:beta"
	newTag := StripImageTagHost(canonicalNewTag)
	err = client.TagImage(testImage, canonicalNewTag)
	if err != nil {
		t.Error(err)
	}

	images, err := client.ListImages()
	if err != nil {
		t.Error(err)
	}

	var hasImage bool
	for _, image := range images {
		for _, tag := range image.Tags {
			spew.Dump(image.Tags)
			if tag == newTag {
				hasImage = true
				break
			}
		}
	}

	if !hasImage {
		t.Error("expected image tag")
	}
}

func TestRemoveImage(t *testing.T) {
	client := NewClient()
	err := client.PullImage(testImage)
	if err != nil {
		t.Error(err)
	}

	err = client.RemoveImage(testImage)
	if err != nil {
		t.Error(err)
	}
}

func TestRemoveAllImages(t *testing.T) {
	client := NewClient()
	err := client.RemoveAllImages()
	if err != nil {
		t.Error(err)
	}

	images, err := client.ListImages()
	if err != nil {
		t.Error(err)
	}

	if len(images) != 0 {
		t.Error("expected number of images to be 0")
	}
}
