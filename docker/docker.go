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
	debug  bool
}

// Config is client config
type Config struct {
	Debug bool
}

// NewClient creates a new client instance
func NewClient(config *Config) *Client {
	return newEnvClient(config)
}

// newEnvClient returns a new client instance based on environment variables
func newEnvClient(config *Config) *Client {
	cl, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("[docker] %s", err)
	}

	return &Client{
		client: cl,
		debug:  config.Debug,
	}
}

// ImageSummary is structure for image summary
type ImageSummary struct {
	ID   string
	Tags []string
	Size int64
}

// ListImages return list of docker images
func (c *Client) ListImages() ([]*ImageSummary, error) {
	images, err := c.client.ImageList(context.Background(), types.ImageListOptions{
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
func (c *Client) HasImage(imageID string) (bool, error) {
	args := filters.NewArgs()
	args.Add("reference", StripImageTagHost(imageID))
	images, err := c.client.ImageList(context.Background(), types.ImageListOptions{
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
func (c *Client) PullImage(imageID string) error {
	reader, err := c.client.ImagePull(context.Background(), imageID, types.ImagePullOptions{})
	if err != nil {
		fmt.Errorf("[docker] error pulling image: %v", err)
		return err
	}

	if c.debug {
		io.Copy(os.Stdout, reader)
	}

	return nil
}

// PushImage pushes a docker image
func (c *Client) PushImage(imageID string) error {
	reader, err := c.client.ImagePush(context.Background(), imageID, types.ImagePushOptions{
		// NOTE: if no auth, then any value is required
		RegistryAuth: "123",
	})
	if err != nil {
		return err
	}

	if c.debug {
		io.Copy(os.Stdout, reader)
	}

	return nil
}

// TagImage tags an image
func (c *Client) TagImage(imageID, tag string) error {
	return c.client.ImageTag(context.Background(), imageID, tag)
}

// RemoveImage remove an image from the local registry
func (c *Client) RemoveImage(imageID string) error {
	_, err := c.client.ImageRemove(context.Background(), imageID, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})

	return err
}

// RemoveAllImages removes all images from the local registry
func (c *Client) RemoveAllImages() error {
	images, err := c.ListImages()
	if err != nil {
		return err
	}

	var lastErr error
	for _, image := range images {
		err := c.RemoveImage(image.ID)
		if err != nil {
			lastErr = err
			continue
		}
	}

	images, err = c.ListImages()
	if err != nil {
		return err
	}

	if len(images) != 0 {
		return lastErr
	}

	return nil
}

// ReadImage reads the contents of an image into an IO reader
func (c *Client) ReadImage(imageID string) (io.Reader, error) {
	return c.client.ImageSave(context.Background(), []string{imageID})
}

// LoadImage loads an image from an IO reader
func (c *Client) LoadImage(input io.Reader) error {
	output, err := c.client.ImageLoad(context.Background(), input, false)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(output.Body)
	c.Debugf("%s", string(body))

	return err
}

// LoadImageByFilePath loads an image from a tarball
func (c *Client) LoadImageByFilePath(filepath string) error {
	input, err := os.Open(filepath)
	if err != nil {
		log.Errorf("[docker] load image by filepath error; %v", err)
		return err
	}
	return c.LoadImage(input)
}

// SaveImageTar saves an image into a tarball
func (c *Client) SaveImageTar(imageID string, dest string) error {
	reader, err := c.ReadImage(imageID)

	fo, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer fo.Close()
	io.Copy(fo, reader)
	return nil
}

// Debugf prints debug log
func (c *Client) Debugf(str string, args ...interface{}) {
	if c.debug {
		log.Printf(str, args...)
	}
}
