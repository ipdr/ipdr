package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	container "github.com/docker/docker/api/types/container"
	log "github.com/sirupsen/logrus"

	types "github.com/docker/docker/api/types"
	filters "github.com/docker/docker/api/types/filters"
	mount "github.com/docker/docker/api/types/mount"
	client "github.com/docker/docker/client"
	nat "github.com/docker/go-connections/nat"
	tlsconfig "github.com/docker/go-connections/tlsconfig"
)

// Client ...
type Client struct {
	client *client.Client
}

// NewClient ...
func NewClient() *Client {
	return newEnvClient()
}

// newClient ...
func newClient() *Client {
	httpclient := &http.Client{}

	if dockerCertPath := os.Getenv("DOCKER_CERT_PATH"); dockerCertPath != "" {
		options := tlsconfig.Options{
			CAFile:             filepath.Join(dockerCertPath, "ca.pem"),
			CertFile:           filepath.Join(dockerCertPath, "cert.pem"),
			KeyFile:            filepath.Join(dockerCertPath, "key.pem"),
			InsecureSkipVerify: os.Getenv("DOCKER_TLS_VERIFY") == "",
		}
		tlsc, err := tlsconfig.Client(options)
		if err != nil {
			log.Fatalf("[docker] %s", err)
		}

		httpclient.Transport = &http.Transport{
			TLSClientConfig: tlsc,
		}
	}

	host := os.Getenv("DOCKER_HOST")
	version := os.Getenv("DOCKER_VERSION")

	if host == "" {
		log.Fatal("[docker] DOCKER_HOST is required")
	}

	if version == "" {
		version = dockerVersionFromCLI()
		if version == "" {
			log.Fatal("[docker] DOCKER_VERSION is required")
		}
	}

	cl, err := client.NewClient(host, version, httpclient, nil)
	if err != nil {
		log.Fatalf("[docker] %s", err)
	}

	return &Client{
		client: cl,
	}
}

// newEnvClient ...
func newEnvClient() *Client {
	cl, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("[docker] %s", err)
	}

	return &Client{
		client: cl,
	}
}

// ImageSummary ....
type ImageSummary struct {
	ID   string
	Tags []string
	Size int64
}

// ListImages ...
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

// HasImage ...
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

// PullImage ...
func (s *Client) PullImage(imageID string) error {
	reader, err := s.client.ImagePull(context.Background(), imageID, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	io.Copy(os.Stdout, reader)
	return nil
}

// PushImage ...
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

// TagImage ...
func (s *Client) TagImage(imageID, tag string) error {
	return s.client.ImageTag(context.Background(), imageID, tag)
}

// RemoveImage ...
func (s *Client) RemoveImage(imageID string) error {
	_, err := s.client.ImageRemove(context.Background(), imageID, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})

	return err
}

// RemoveAllImages ...
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

// CreateContainerConfig ...
type CreateContainerConfig struct {
	// container:host
	Volumes map[string]string
	Ports   map[string]string
}

// CreateContainer ...
func (s *Client) CreateContainer(imageID string, cmd []string, config *CreateContainerConfig) (string, error) {
	if config == nil {
		config = &CreateContainerConfig{}
	}

	dockerConfig := &container.Config{
		Image:        imageID,
		Cmd:          cmd,
		Tty:          false,
		Volumes:      map[string]struct{}{},
		ExposedPorts: map[nat.Port]struct{}{},
	}

	hostConfig := &container.HostConfig{
		Binds:        nil,
		PortBindings: map[nat.Port][]nat.PortBinding{},
		AutoRemove:   true,
		IpcMode:      "",
		Privileged:   false,
		Mounts:       []mount.Mount{},
	}

	if len(config.Volumes) > 0 {
		for k, v := range config.Volumes {
			dockerConfig.Volumes[k] = struct{}{}
			hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
				Type:     "bind",
				Source:   v,
				Target:   k,
				ReadOnly: false,
			})
		}
	}

	if len(config.Ports) > 0 {
		for k, v := range config.Ports {
			t, err := nat.NewPort("tcp", k)
			if err != nil {
				return "", err
			}
			dockerConfig.ExposedPorts[t] = struct{}{}
			hostConfig.PortBindings[t] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: v,
				},
			}
		}
	}

	resp, err := s.client.ContainerCreate(context.Background(), dockerConfig, hostConfig, nil, "")
	if err != nil {
		return "", err
	}

	log.Printf("[docker] container %s is ready", resp.ID)

	return resp.ID, nil
}

// StartContainer ...
func (s *Client) StartContainer(containerID string) error {
	err := s.client.ContainerStart(context.TODO(), containerID, types.ContainerStartOptions{})

	if err != nil {
		log.Printf("[docker] error starting container %s; %v", containerID, err)
		return err
	}

	reader, err := s.client.ContainerLogs(context.TODO(), containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})

	go func() {
		log.Println("[docker] logging to stdout..")
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("[docker] [container %s] [log] %s\n", ShortContainerID(containerID), line)
		}
	}()

	if err != nil {
		log.Printf("[docker] error reading container logs for %s; %v", containerID, err)
		return err
	}

	log.Printf("[docker] running container %s", containerID)
	return nil
}

// StopContainer ...
func (s *Client) StopContainer(containerID string) error {
	log.Printf("[docker] stopping container %s", containerID)
	err := s.client.ContainerStop(context.Background(), containerID, nil)
	if err != nil {
		return err
	}

	log.Println("[docker] container stopped")
	return nil
}

// InspectContainer ...
func (s *Client) InspectContainer(containerID string) (types.ContainerJSON, error) {
	info, err := s.client.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return types.ContainerJSON{}, err
	}

	return info, nil
}

// ContainerExec ...
func (s *Client) ContainerExec(containerID string, cmd []string) (io.Reader, error) {
	id, err := s.client.ContainerExecCreate(context.Background(), containerID, types.ExecConfig{
		AttachStdout: true,
		Cmd:          cmd,
	})

	log.Printf("[docker] exec ID %s", id.ID)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.ContainerExecAttach(context.Background(), id.ID, types.ExecConfig{})
	if err != nil {
		return nil, err
	}

	return resp.Reader, nil
}

// ReadImage ...
func (s *Client) ReadImage(imageID string) (io.Reader, error) {
	return s.client.ImageSave(context.Background(), []string{imageID})
}

// LoadImage ...
func (s *Client) LoadImage(input io.Reader) error {
	output, err := s.client.ImageLoad(context.Background(), input, false)
	if err != nil {
		return err
	}

	//io.Copy(os.Stdout, output)
	fmt.Println(output)
	body, err := ioutil.ReadAll(output.Body)
	fmt.Println(string(body))

	return err
}

// LoadImageByFilepath ...
func (s *Client) LoadImageByFilepath(filepath string) error {
	input, err := os.Open(filepath)
	if err != nil {
		log.Printf("[docker] load image by filepath error; %v", err)
		return err
	}
	return s.LoadImage(input)
}

// SaveImageTar ...
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

func dockerVersionFromCLI() string {
	cmd := `docker version --format="{{.Client.APIVersion}}"`
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

// CommitContainer ...
func (s *Client) CommitContainer(containerID string) (string, error) {
	resp, err := s.client.ContainerCommit(context.Background(), containerID, types.ContainerCommitOptions{})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}
