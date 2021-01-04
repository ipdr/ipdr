package registry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	docker "github.com/miguelmota/ipdr/docker"
	ipfs "github.com/miguelmota/ipdr/ipfs"
	netutil "github.com/miguelmota/ipdr/netutil"
	server "github.com/miguelmota/ipdr/server"
	log "github.com/sirupsen/logrus"
)

// Registry is the registry structure
type Registry struct {
	dockerLocalRegistryHost string
	dockerClient            *docker.Client
	ipfsClient              *ipfs.Client
	debug                   bool
}

// Config is the config for the registry
type Config struct {
	DockerLocalRegistryHost string
	IPFSHost                string
	IPFSGateway             string
	Debug                   bool
}

// NewRegistry returns a new registry client instance
func NewRegistry(config *Config) *Registry {
	if config == nil {
		config = &Config{}
	}

	dockerLocalRegistryHost := config.DockerLocalRegistryHost
	if dockerLocalRegistryHost == "" {
		dockerLocalRegistryHost = os.Getenv("DOCKER_LOCAL_REGISTRY_HOST")
		if dockerLocalRegistryHost == "" {
			localIP, err := netutil.LocalIP()
			if err != nil {
				log.Fatalf("[registry] %s", err)
			}

			dockerLocalRegistryHost = localIP.String()
		}
	}

	ipfsClient := ipfs.NewRemoteClient(&ipfs.Config{
		Host:       config.IPFSHost,
		GatewayURL: config.IPFSGateway,
	})
	dockerClient := docker.NewClient(&docker.Config{
		Debug: config.Debug,
	})

	return &Registry{
		dockerLocalRegistryHost: dockerLocalRegistryHost,
		ipfsClient:              ipfsClient,
		dockerClient:            dockerClient,
		debug:                   config.Debug,
	}
}

// PushImageByID uploads Docker image by image ID, which is hash or repo tag, to IPFS
func (r *Registry) PushImageByID(imageID string) (string, error) {
	// normalize image ID
	id, err := r.TagToImageID(imageID)
	if err != nil {
		return "", err
	}

	reader, err := r.dockerClient.ReadImage(id)
	if err != nil {
		return "", err
	}

	return r.PushImage(reader, imageID)
}

// TagToImageID returns the image ID given a repo tag
func (r *Registry) TagToImageID(imageID string) (string, error) {
	images, err := r.dockerClient.ListImages()
	if err != nil {
		return "", err
	}

	for _, image := range images {
		if strings.HasPrefix(image.ID, imageID) {
			break
		}
		for _, tag := range image.Tags {
			if tag == imageID || tag == imageID+":latest" {
				imageID = image.ID
			}
		}
	}

	return imageID, nil
}

// PushImage uploads the Docker image to IPFS
func (r *Registry) PushImage(reader io.Reader, imageID string) (string, error) {
	tmp, err := mktmp()
	if err != nil {
		return "", err
	}

	r.Debugf("[registry] temp: %s", tmp)
	if err := untar(reader, tmp); err != nil {
		return "", err
	}

	root, err := r.ipfsPrep(tmp, imageID)
	if err != nil {
		return "", err
	}

	r.Debugf("[registry] root dir: %s", root)
	imageIpfsHash, err := r.uploadDir(root)
	if err != nil {
		return "", err
	}

	r.Debugf("\n[registry] uploaded to /ipfs/%s\n", imageIpfsHash)
	r.Debugf("[registry] docker image %s\n", imageIpfsHash)

	return imageIpfsHash, nil
}

// DownloadImage downloads the Docker image from IPFS
func (r *Registry) DownloadImage(ipfsHash string) (string, error) {
	tmp, err := mktmp()
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/%s.tar", tmp, ipfsHash)
	err = r.ipfsClient.Get(ipfsHash, path)
	if err != nil {
		return "", err
	}

	return path, nil
}

// PullImage pulls the Docker image from IPFS
func (r *Registry) PullImage(ipfsHash string) (string, error) {
	r.runServer()
	// dockerizedHash := regutil.DockerizeHash(ipfsHash)
	dockerPullImageID := fmt.Sprintf("%s/%s", r.dockerLocalRegistryHost, ipfsHash)

	r.Debugf("[registry] attempting to pull %s", dockerPullImageID)
	err := r.dockerClient.PullImage(dockerPullImageID)
	if err != nil {
		log.Errorf("[registry] error pulling image %s; %v", dockerPullImageID, err)
		return "", err
	}

	return dockerPullImageID, nil
}

// retag retags an image
func (r *Registry) retag(dockerPullImageID, dockerizedHash string) error {
	err := r.dockerClient.TagImage(dockerPullImageID, dockerizedHash)
	if err != nil {
		log.Errorf("[registry] error tagging image %s; %v", dockerizedHash, err)
		return err
	}

	r.Debugf("[registry] tagged image as %s", dockerizedHash)

	err = r.dockerClient.RemoveImage(dockerPullImageID)
	if err != nil {
		log.Errorf("[registry] error removing image %s; %v", dockerPullImageID, err)
		return err
	}

	return nil
}

func (r *Registry) runServer() {
	timeout := time.Duration(100 * time.Millisecond)
	client := http.Client{
		Timeout: timeout,
	}
	url := fmt.Sprintf("http://%s/health", r.dockerLocalRegistryHost)
	resp, err := client.Get(url)
	if err != nil || resp.StatusCode != 200 {
		srv := server.NewServer(&server.Config{
			Port:        netutil.ExtractPort(r.dockerLocalRegistryHost),
			Debug:       r.debug,
			IPFSGateway: r.ipfsClient.GatewayURL(),
		})
		go srv.Start()
	}
}

// ipfsPrep formats the image data into a registry compatible format
func (r *Registry) ipfsPrep(tmp string, imageID string) (string, error) {
	root, err := mktmp()
	if err != nil {
		return "", err
	}

	workdir := root
	r.Debugf("[registry] preparing image in: %s", workdir)
	name := "default"

	// read human readable name of image
	if _, err := os.Stat(tmp + "repositories"); err == nil {
		reposJSON, err := readJSON(tmp + "/repositories")
		if err != nil {
			return "", err
		}
		if len(reposJSON) != 1 {
			return "", errors.New("only one repository expected in input file")
		}
		for imageName, tags := range reposJSON {
			r.Debugf("[registry] %s %s", imageName, tags)
			if len(tags) != 1 {
				return "", fmt.Errorf("only one tag expected for %s", imageName)
			}
			for tag, hash := range tags {
				name = normalizeImageName(imageName)
				r.Debugf("[registry] processing image:%s tag:%s hash:256:%s", name, tag, hash)
			}
		}
	}

	workdir = workdir + "/" + name
	mkdir(workdir)
	mkdir(workdir + "/manifests")
	mkdir(workdir + "/blobs")
	manifestJSON, err := readJSONArray(tmp + "/manifest.json")
	if err != nil {
		return "", err
	}

	if len(manifestJSON) == 0 {
		return "", errors.New("expected manifest to contain data")
	}

	manifest := manifestJSON[0]
	configFile, ok := manifest["Config"].(string)
	if !ok {
		return "", errors.New("image archive must be produced by docker > 1.10")
	}

	configDigest := "sha256:" + string(configFile[:len(configFile)-5])
	configDest := fmt.Sprintf("%s/blobs/%s", workdir, configDigest)
	r.Debugf("\n[registry] dist: %s", configDest)

	if err := copyFile(tmp+"/"+configFile, configDest); err != nil {
		return "", err
	}

	mf, err := r.makeV2Manifest(manifest, configDigest, configDest, tmp, workdir)
	if err != nil {
		return "", err
	}

	ref := func(s string) string {
		//name:tag
		//sha256:hex
		if strings.Index(s, "sha256:") != -1 {
			return "latest"
		}
		ss := strings.SplitN(s, ":", 2)
		if len(ss) == 2 {
			return ss[1]
		}
		return "latest"
	}
	writeManifest := func() error {
		tag := ref(imageID)
		if tag != "latest" {
			if err = writeJSON(mf, workdir+"/manifests/latest"); err != nil {
				return err
			}
		}
		if err = writeJSON(mf, workdir+"/manifests/"+tag); err != nil {
			return err
		}
		data, err := json.Marshal(mf)
		if err != nil {
			return err
		}
		rd := sha256.Sum256(data)
		return writeJSON(mf, workdir+"/manifests/sha256:"+hex.EncodeToString(rd[:]))
	}
	if err := writeManifest(); err != nil {
		return "", err
	}

	return root, nil
}

// uploadDir uploads the directory to IPFS
func (r *Registry) uploadDir(root string) (string, error) {
	hash, err := r.ipfsClient.AddDir(root)
	if err != nil {
		return "", err
	}

	r.Debugf("[registry] upload hash %s", hash)

	// get the first ref, which contains the image data
	refs, err := r.ipfsClient.Refs(hash, false)
	if err != nil {
		return "", err
	}

	var firstRef string
	for i := 0; i < 10; i++ {
		firstRef = <-refs

		if firstRef != "" {
			return firstRef, nil
		}
	}

	// return base hash if no refs
	if firstRef == "" {
		log.Fatal("NO REF")
		return hash, nil
	}

	return "", errors.New("could not upload")
}

// mktmp creates a temporary directory
func mktmp() (string, error) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	return tmp, err
}

// Debugf prints debug log
func (r *Registry) Debugf(str string, args ...interface{}) {
	if r.debug {
		log.Printf(str, args...)
	}
}

// ipfsShellCmd executes an IPFS command via the shell
func ipfsShellCmd(cmdStr string) (string, string, error) {
	path, err := exec.LookPath("ipfs")
	if err != nil {
		return "", "", errors.New("ipfs command was not found. Please install ipfs")
	}
	cmd := exec.Command("sh", "-c", fmt.Sprintf("%s %s", path, cmdStr))
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()
	var stdoutBuf, stderrBuf bytes.Buffer
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err = cmd.Start()
	if err != nil {
		return "", "", err
	}

	go copyio(stdoutIn, stdout)
	go copyio(stderrIn, stderr)

	err = cmd.Wait()
	if err != nil {
		return "", "", err
	}

	outstr := strings.TrimSpace(string(stdoutBuf.Bytes()))
	errstr := strings.TrimSpace(string(stderrBuf.Bytes()))

	return outstr, errstr, nil
}

// copyio is a helper to copy IO readers
func copyio(out io.Reader, in io.Writer) error {
	_, err := io.Copy(in, out)
	if err != nil {
		return err
	}

	return nil
}

// writeJSON writes an interface to a JSON file
func writeJSON(idate interface{}, path string) error {
	data, err := json.Marshal(idate)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, data, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// produce v2 manifest of type/application/vnd.docker.distribution.manifest.v2+json
func (r *Registry) makeV2Manifest(manifest map[string]interface{}, configDigest, configDest, tmp, workdir string) (map[string]interface{}, error) {
	v2manifest, err := r.prepareV2Manifest(manifest, tmp, workdir+"/blobs")
	if err != nil {
		return nil, err
	}
	config := make(map[string]interface{})
	config["digest"] = configDigest
	config["size"], err = fileSize(configDest)
	if err != nil {
		return nil, err
	}
	config["mediaType"] = "application/vnd.docker.container.image.v1+json"
	conf, ok := v2manifest["config"].(map[string]interface{})
	if !ok {
		return nil, errors.New("not ok")
	}
	v2manifest["config"] = mergemap(conf, config)
	return v2manifest, nil
}

// mergemap merges two maps
func mergemap(a, b map[string]interface{}) map[string]interface{} {
	for k, v := range b {
		a[k] = v
	}
	return a
}

// prepareV2Manifest preps the docker image into a docker registry V2 manifest format
func (r *Registry) prepareV2Manifest(mf map[string]interface{}, tmp, blobDir string) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	res["schemaVersion"] = 2
	res["mediaType"] = "application/vnd.docker.distribution.manifest.v2+json"
	config := make(map[string]interface{})
	res["config"] = config
	var layers []map[string]interface{}
	mediaType := "application/vnd.docker.image.rootfs.diff.tar.gzip"
	ls, ok := mf["Layers"].([]interface{})
	if !ok {
		return nil, errors.New("expected layers")
	}
	for _, ifc := range ls {
		layer, ok := ifc.(string)
		if !ok {
			return nil, errors.New("expected string")
		}
		obj := make(map[string]interface{})
		obj["mediaType"] = mediaType
		size, digest, err := r.compressLayer(tmp+"/"+layer, blobDir)
		if err != nil {
			return nil, err
		}
		obj["size"] = size
		obj["digest"] = "sha256:" + digest
		layers = append(layers, obj)
	}
	res["layers"] = layers
	return res, nil
}

// compressLayer returns the sha256 hash of a directory
func (r *Registry) compressLayer(path, blobDir string) (int64, string, error) {
	r.Debugf("[registry] compressing layer: %s", path)
	tmp := blobDir + "/layer.tmp.tgz"

	err := gzipFile(path, tmp)
	if err != nil {
		return int64(0), "", err
	}

	digest, err := sha256File(tmp)
	if err != nil {
		return int64(0), "", err
	}

	size, err := fileSize(tmp)
	if err != nil {
		return int64(0), "", err
	}

	err = renameFile(tmp, fmt.Sprintf("%s/sha256:%s", blobDir, digest))
	if err != nil {
		return int64(0), "", err
	}

	return size, digest, nil
}

// gzipFile gzips a file into a destination file
func gzipFile(src, dst string) error {
	data, _ := ioutil.ReadFile(src)
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(data)
	w.Close()

	return ioutil.WriteFile(dst, b.Bytes(), os.ModePerm)
}

// fileSize returns the size of the file
func fileSize(path string) (int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return int64(0), err
	}

	return fi.Size(), nil
}

// sha256 returns the sha256 hash of a file
func sha256File(path string) (string, error) {
	// TODO: stream instead of reading whole image in memory
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()

	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// renameFile renames a file
func renameFile(src, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		return err
	}

	return nil
}

// mkdir creates a directory if it doesn't exist
func mkdir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}
}

func copyFile(src, dst string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, data, 0644)
}

// untar untars a reader into a destination directory
func untar(reader io.Reader, dst string) error {
	tr := tar.NewReader(reader)

	for {
		header, err := tr.Next()
		switch {
		// no more files
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {
		// create directory if doesn't exit
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		// create file
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer f.Close()

			// copy contents to file
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
		}
	}
}

// readJSON reads a file into a map structure
func readJSON(filepath string) (map[string]map[string]string, error) {
	body, _ := ioutil.ReadFile(filepath)
	var data map[string]map[string]string
	err := json.Unmarshal(body, &data)
	if err != nil {
		return data, err
	}

	return data, nil
}

// readJSONArray reads a file into an array of map structures
func readJSONArray(filepath string) ([]map[string]interface{}, error) {
	body, _ := ioutil.ReadFile(filepath)
	var data []map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		return data, err
	}

	return data, nil
}

// normalizeImageName normalizes an image name
func normalizeImageName(name string) string {
	// TODO
	return name
}
