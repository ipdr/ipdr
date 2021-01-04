package ipfs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	api "github.com/ipfs/go-ipfs-api"
	files "github.com/ipfs/go-ipfs-files"
	log "github.com/sirupsen/logrus"
)

// Client is the client structure
type Client struct {
	client     *api.Shell
	isRemote   bool
	host       string
	gatewayURL string
}

// Config is the config for the client
type Config struct {
	Host       string
	GatewayURL string
}

// NewClient returns a new IPFS client instance
func NewClient() *Client {
	err := RunDaemon()
	if err != nil {
		log.Fatalf("[ipfs] %s", err)
	}

	url, err := getIpfsAPIURL()
	if err != nil {
		log.Fatalf("[ipfs] %s", err)
	}

	client := api.NewShell(url)
	return &Client{
		client: client,
		host:   url,
	}
}

// NewRemoteClient returns a new IPFS shell client
func NewRemoteClient(config *Config) *Client {
	if config == nil {
		config = &Config{}
	}

	client := api.NewShell(config.Host)
	host := config.Host
	if host == "" {
		var err error
		host, err = getIpfsAPIURL()
		if err != nil {
			log.Fatal(err)
		}
	}

	return &Client{
		client:     client,
		isRemote:   true,
		host:       host,
		gatewayURL: config.GatewayURL,
	}
}

// Cat the content at the given path. Callers need to drain and close the returned reader after usage.
func (client *Client) Cat(path string) (io.ReadCloser, error) {
	return client.client.Cat(path)
}

// Get fetches the contents and outputs into a directory
func (client *Client) Get(hash, outdir string) error {
	return client.client.Get(hash, outdir)
}

// List entries at the given path
func (client *Client) List(path string) ([]*api.LsLink, error) {
	return client.client.List(path)
}

// AddDir adds a directory to IPFS
// https://github.com/ipfs/go-ipfs-api/blob/master/add.go#L99-L145
func (client *Client) AddDir(dir string) (string, error) {
	stat, err := os.Lstat(dir)
	if err != nil {
		return "", err
	}

	sf, err := files.NewSerialFile(dir, false, stat)
	if err != nil {
		return "", err
	}
	slf := files.NewSliceDirectory([]files.DirEntry{files.FileEntry(filepath.Base(dir), sf)})
	reader := files.NewMultiFileReader(slf, true)

	resp, err := client.client.Request("add").
		Option("recursive", true).
		Option("cid-version", 1).
		Body(reader).
		Send(context.Background())
	if err != nil {
		return "", nil
	}

	defer resp.Close()

	if resp.Error != nil {
		return "", resp.Error
	}

	dec := json.NewDecoder(resp.Output)
	var final string
	for {
		var out object
		err = dec.Decode(&out)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		final = out.Hash
	}

	if final == "" {
		return "", errors.New("no results received")
	}

	return final, nil
}

// Refs returns the refs of an IPFS hash
func (client *Client) Refs(hash string, recursive bool) (<-chan string, error) {
	if client.isRemote {
		return client.remoteRefs(hash, recursive)
	}

	return client.client.Refs(hash, recursive)
}

// GatewayURL returns the gateway URL
func (client *Client) GatewayURL() string {
	if client.gatewayURL == "" {
		url, err := HostGatewayURL()
		if err == nil {
			return url
		}
	}

	return NormalizeGatewayURL(client.gatewayURL)
}

// remoteRefs returns refs using the IPFS API
// https://docs.ipfs.io/reference/http/api/#api-v0-refs
func (client *Client) remoteRefs(hash string, recursive bool) (<-chan string, error) {
	url := fmt.Sprintf("http://%s/api/v0/refs?arg=%s&max-depth=%s", client.host, hash, map[bool]string{true: "-1", false: "1"}[recursive])

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error: %s", resp.Status)
	}

	out := make(chan string)
	go func() {
		defer resp.Body.Close()
		defer close(out)

		var ref struct {
			Ref string
		}
		dec := json.NewDecoder(resp.Body)

		for {
			err := dec.Decode(&ref)
			if err != nil {
				return
			}
			if len(ref.Ref) > 0 {
				out <- ref.Ref
			}
		}
	}()

	return out, nil
}

type object struct {
	Hash string
}

// AddImage adds components of an image recursively
func (client *Client) AddImage(manifest map[string][]byte, layers map[string][]byte) (string, error) {
	mf := make(map[string]files.Node)
	for k, v := range manifest {
		mf[k] = files.NewBytesFile(v)
	}

	bf := make(map[string]files.Node)
	for k, v := range layers {
		bf[k] = files.NewBytesFile(v)
	}

	sf := files.NewMapDirectory(map[string]files.Node{
		"blobs":     files.NewMapDirectory(bf),
		"manifests": files.NewMapDirectory(mf),
	})
	slf := files.NewSliceDirectory([]files.DirEntry{files.FileEntry("image", sf)})

	reader := files.NewMultiFileReader(slf, true)
	resp, err := client.client.Request("add").
		Option("recursive", true).
		Option("cid-version", 1).
		Body(reader).
		Send(context.Background())
	if err != nil {
		return "", nil
	}

	defer resp.Close()

	if resp.Error != nil {
		return "", resp.Error
	}

	dec := json.NewDecoder(resp.Output)
	var final string
	for {
		var out object
		err = dec.Decode(&out)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		final = out.Hash
	}

	if final == "" {
		return "", errors.New("no results received")
	}

	return final, nil
}

// RunDaemon runs the IPFS daemon
func RunDaemon() error {
	var err error
	ready := make(chan bool)
	go func() {
		if err = spawnIpfsDaemon(ready); err != nil {
			log.Errorf("[ipfs] %s", err)
		}
	}()

	if !<-ready {
		return errors.New("failed to start IPFS daemon")
	}

	return nil
}

// spawnIpfsDaemon spawns the IPFS daemon by issuing shell commands
func spawnIpfsDaemon(ready chan bool) error {
	out, err := exec.Command("pgrep", "ipfs").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		log.Warn("[ipfs] IPFS is not running. Starting...")

		go func() {
			// TODO: detect when running by watching log output
			time.Sleep(10 * time.Second)
			ready <- true
		}()

		err := exec.Command("ipfs", "daemon").Run()
		if err != nil {
			ready <- false
			log.Errorf("[ipfs] %s", err)
			return errors.New("failed to start IPFS")
		}
	}

	ready <- true
	//log.Println("[ipfs] IPFS is running...")
	return nil
}

// NormalizeGatewayURL normalizes IPFS gateway URL
func NormalizeGatewayURL(urlstr string) string {
	if !strings.HasPrefix(urlstr, "http") {
		urlstr = "http://" + urlstr
	}
	u, err := url.Parse(urlstr)
	if err != nil {
		panic(err)
	}

	scheme := u.Scheme
	if u.Scheme != "" {
		scheme = "http"
	}

	host := u.Hostname()
	if host == "" {
		host = "ipfs.io"
	}

	var user string
	if u.User != nil {
		user = u.User.String() + "@"
	}

	port := u.Port()
	if port != "" {
		port = ":" + port
	}

	return fmt.Sprintf("%s://%s%s%s", scheme, user, host, port)
}

// HostGatewayURL returns IPFS gateway URL that host is configured to use
func HostGatewayURL() (string, error) {
	port, err := getIpfsGatewayPort()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("http://127.0.0.1:%s", port), nil
}

// getIpfsAPIURL returns the IPFS API base URL
func getIpfsAPIURL() (string, error) {
	out, err := exec.Command("ipfs", "config", "Addresses.API").Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// getIpfsGatewayPort return the IPFS gateway port number
func getIpfsGatewayPort() (string, error) {
	out, err := exec.Command("ipfs", "config", "Addresses.Gateway").Output()
	if err != nil {
		return "", err
	}

	ipld := strings.TrimSpace(string(out))
	parts := strings.Split(ipld, "/")

	if len(parts) == 0 {
		return "", errors.New("[ipfs] gateway config not found")
	}

	return parts[len(parts)-1], nil
}
