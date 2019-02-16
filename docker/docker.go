package docker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"

	types "github.com/docker/docker/api/types"
	filters "github.com/docker/docker/api/types/filters"
	client "github.com/docker/docker/client"
)

// Client is client structure
type Client struct {
	client *client.Client
}

// NewClient creates a new client instance
func NewClient() *Client {
	return newEnvClient()
}

// newEnvClient returns a new client instance based on environment variables
func newEnvClient() *Client {
	cl, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("[docker] %s", err)
	}

	return &Client{
		client: cl,
	}
}

// ImageSummary is structure for image summary
type ImageSummary struct {
	ID   string
	Tags []string
	Size int64
}

// ListImages return list of docker images
func (s *Client) ListImages() ([]*ImageSummary, error) {
	images, err := s.client.ImageList(context.Background(), types.ImageListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	var summaries []*ImageSummary
	for _, image := range images {
		summaries = append(summaries, &ImageSummary{
			ID:   image.ID,
			Tags: image.RepoTags,
			Size: image.Size,
		})
	}

	return summaries, nil
}

// HasImage returns true if image ID is available locally
func (s *Client) HasImage(imageID string) (bool, error) {
	args := filters.NewArgs()
	args.Add("reference", StripImageTagHost(imageID))
	images, err := s.client.ImageList(context.Background(), types.ImageListOptions{
		All:     true,
		Filters: args,
	})
	if err != nil {
		return false, err
	}

	if len(images) > 0 {
		return true, nil
	}

	return false, nil
}

// PullImage pulls a docker image
func (s *Client) PullImage(imageID string) error {
	reader, err := s.client.ImagePull(context.Background(), imageID, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	io.Copy(os.Stdout, reader)
	return nil
}

// PushImage pushes a docker image
func (s *Client) PushImage(imageID string) error {
	reader, err := s.client.ImagePush(context.Background(), imageID, types.ImagePushOptions{
		// NOTE: if no auth, then any value is required
		RegistryAuth: "123",
	})
	if err != nil {
		return err
	}
	io.Copy(os.Stdout, reader)
	return nil
}

// TagImage tags an image
func (s *Client) TagImage(imageID, tag string) error {
	return s.client.ImageTag(context.Background(), imageID, tag)
}

// RemoveImage remove an image from the local registry
func (s *Client) RemoveImage(imageID string) error {
	_, err := s.client.ImageRemove(context.Background(), imageID, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})

	return err
}

// RemoveAllImages removes all images from the local registry
func (s *Client) RemoveAllImages() error {
	images, err := s.ListImages()
	if err != nil {
		return err
	}

	var lastErr error
	for _, image := range images {
		err := s.RemoveImage(image.ID)
		if err != nil {
			lastErr = err
			continue
		}
	}

	images, err = s.ListImages()
	if err != nil {
		return err
	}

	if len(images) != 0 {
		return lastErr
	}

	return nil
}

// ReadImage reads the contents of an image into an IO reader
func (s *Client) ReadImage(imageID string) (io.Reader, error) {
	return s.client.ImageSave(context.Background(), []string{imageID})
}

// LoadImage loads an image from an IO reader
func (s *Client) LoadImage(input io.Reader) error {
	output, err := s.client.ImageLoad(context.Background(), input, false)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(output.Body)
	fmt.Println(string(body))

	return err
}

// LoadImageByFilePath loads an image from a tarball
func (s *Client) LoadImageByFilePath(filepath string) error {
	input, err := os.Open(filepath)
	if err != nil {
		log.Printf("[docker] load image by filepath error; %v", err)
		return err
	}
	return s.LoadImage(input)
}

// SaveImageTar saves an image into a tarball
func (s *Client) SaveImageTar(imageID string, dest string) error {
	reader, err := s.ReadImage(imageID)

	fo, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer fo.Close()
	io.Copy(fo, reader)
	return nil
}
