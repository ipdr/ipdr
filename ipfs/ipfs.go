package ipfs

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	api "github.com/ipfs/go-ipfs-api"
	log "github.com/sirupsen/logrus"
)

// Client ...
type Client struct {
	client   *api.Shell
	isRemote bool
	host     string
}

// NewClient ...
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
	}
}

// NewRemoteClient ...
func NewRemoteClient(host string) *Client {
	client := api.NewShell(host)
	return &Client{
		client:   client,
		isRemote: true,
		host:     host,
	}
}

// Get ...
func (client *Client) Get(hash, outdir string) error {
	return client.client.Get(hash, outdir)
}

// AddDir ...
func (client *Client) AddDir(dir string) (string, error) {
	return client.client.AddDir(dir)
}

// Refs ...
func (client *Client) Refs(hash string, recursive bool) (<-chan string, error) {
	if client.isRemote {
		return client.remoteRefs(hash, recursive)
	}

	return client.client.Refs(hash, recursive)
}

func (client *Client) remoteRefs(hash string, recursive bool) (<-chan string, error) {
	url := fmt.Sprintf("http://%s/api/v0/refs?arg=%s&recursive=%v", client.host, hash, recursive)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	var ref struct {
		Ref string
	}

	out := make(chan string)
	dec := json.NewDecoder(resp.Body)
	go func() {
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

// RunDaemon ...
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

// hacky way to spawn daemon
// TODO: improve
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
	log.Println("[ipfs] IPFS is running...")
	return nil
}

// GatewayURL returns IPFS gateway URL
func GatewayURL() (string, error) {
	port, err := getIpfsGatewayPort()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("http://127.0.0.1:%s", port), nil
}

func getIpfsAPIURL() (string, error) {
	out, err := exec.Command("ipfs", "config", "Addresses.API").Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

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
