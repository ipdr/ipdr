package docker

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"

	types "github.com/docker/docker/api/types"
	dclient "github.com/docker/docker/client"
)

var (
	testImage    = "docker.io/library/alpine"
	testImageTar = "hello-world.tar"
)

func init() {
	createTestTar()
}

func TestNew(t *testing.T) {
	client := createClient()
	if client == nil {
		t.Error("expected instance")
	}
}

func TestListImages(t *testing.T) {
	client := createClient()
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
	client := createClient()
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
	client := createClient()
	err := client.PullImage(testImage)
	if err != nil {
		t.Error(err)
	}
}

func TestReadImage(t *testing.T) {
	client := createClient()
	err := client.PullImage(testImage)
	if err != nil {
		t.Error(err)
	}
	reader, err := client.ReadImage(testImage)
	if err != nil {
		t.Error(err)
	}

	io.Copy(ioutil.Discard, reader)
}

func TestLoadImage(t *testing.T) {
	client := createClient()
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
	client := createClient()
	err := client.LoadImageByFilePath(testImageTar)
	if err != nil {
		t.Error(err)
	}
}

func TestTagImage(t *testing.T) {
	client := createClient()
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
	client := createClient()
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
	t.Skip("Skipping TestRemoveAllImages... Comment skip call to run test. Caution it will remove all images.")
	client := createClient()
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

// last function to run so it cleans up
// the generated test files
func TestCleanup(t *testing.T) {
	cleanUp()
}

func createClient() *Client {
	return NewClient(nil)
}

func createTestTar() {
	ctx := context.Background()
	cli, err := dclient.NewClientWithOpts(dclient.FromEnv)
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(ctx)

	pullR, err := cli.ImagePull(ctx, testImage, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}

	io.Copy(ioutil.Discard, pullR)

	saveR, err := cli.ImageSave(ctx, []string{testImage})

	fo, err := os.Create(testImageTar)
	if err != nil {
		panic(err)
	}

	defer fo.Close()

	io.Copy(fo, saveR)
}

func cleanUp() {
	os.Remove(testImageTar)
}
