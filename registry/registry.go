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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	docker "github.com/miguelmota/ipdr/docker"
	ipfs "github.com/miguelmota/ipdr/ipfs"
	netutil "github.com/miguelmota/ipdr/netutil"
	regutil "github.com/miguelmota/ipdr/regutil"
	server "github.com/miguelmota/ipdr/server"
	log "github.com/sirupsen/logrus"
)

// Registry ...
type Registry struct {
	dockerLocalRegistryHost string
	ipfsClient              *ipfs.Client
}

// Config ...
type Config struct {
	DockerLocalRegistryHost string
	IPFSHost                string
}

// NewRegistry ...
func NewRegistry(config *Config) *Registry {
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

	ipfsHost := "127.0.0.1:5001"
	if config.IPFSHost != "" {
		ipfsHost = config.IPFSHost
	}

	ipfsClient := ipfs.NewRemoteClient(ipfsHost)

	return &Registry{
		dockerLocalRegistryHost: dockerLocalRegistryHost,
		ipfsClient:              ipfsClient,
	}
}

// PushImageByID uploads Docker image by image ID (hash or repo tag) to IPFS
func (registry *Registry) PushImageByID(imageID string) (string, error) {
	// normalize image ID
	imageID, err := registry.TagToImageID(imageID)
	if err != nil {
		return "", err
	}

	client := docker.NewClient()
	reader, err := client.ReadImage(imageID)
	if err != nil {
		return "", err
	}

	return registry.PushImage(reader)
}

// TagToImageID returns the image ID given a repo tag
func (registry *Registry) TagToImageID(imageID string) (string, error) {
	client := docker.NewClient()
	images, err := client.ListImages()
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

// PushImage uploads Docker image to IPFS
func (registry *Registry) PushImage(reader io.Reader) (string, error) {
	tmp, err := mktmp()
	if err != nil {
		return "", err
	}

	log.Printf("[registry] temp: %s", tmp)

	if err := untar(reader, tmp); err != nil {
		return "", err
	}

	root, err := ipfsPrep(tmp)
	if err != nil {
		return "", err
	}

	log.Printf("[registry] root dir: %s", root)

	imageIpfsHash, err := registry.uploadDir(root)
	if err != nil {
		return "", err
	}

	log.Printf("\n[registry] uploaded to /ipfs/%s\n", imageIpfsHash)
	log.Printf("[registry] docker image %s\n", regutil.DockerizeHash(imageIpfsHash))

	return imageIpfsHash, nil
}

// DownloadImage download Docker image from IPFS
func (registry *Registry) DownloadImage(ipfsHash string) (string, error) {
	tmp, err := mktmp()
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/%s.tar", tmp, ipfsHash)
	err = registry.ipfsClient.Get(ipfsHash, path)
	if err != nil {
		return "", err
	}

	return path, nil
}

// PullImage pull Docker image from IPFS
func (registry *Registry) PullImage(ipfsHash string) (string, error) {
	go server.Run()
	client := docker.NewClient()

	dockerizedHash := regutil.DockerizeHash(ipfsHash)
	dockerPullImageID := fmt.Sprintf("%s:%v/%s", registry.dockerLocalRegistryHost, 5000, dockerizedHash)

	log.Printf("[registry] attempting to pull %s", dockerPullImageID)
	err := client.PullImage(dockerPullImageID)
	if err != nil {
		log.Printf("[registry] error pulling image %s; %v", dockerPullImageID, err)
		return "", err
	}

	err = client.TagImage(dockerPullImageID, dockerizedHash)
	if err != nil {
		log.Printf("[registry] error tagging image %s; %v", dockerizedHash, err)
		return "", err
	}

	log.Printf("[registry] tagged image as %s", dockerizedHash)

	err = client.RemoveImage(dockerPullImageID)
	if err != nil {
		log.Printf("[registry] error removing image %s; %v", dockerPullImageID, err)
		return "", err
	}

	return dockerizedHash, nil
}

func mktmp() (string, error) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	return tmp, err
}

func ipfsPrep(tmp string) (string, error) {
	root, err := mktmp()
	if err != nil {
		return "", err
	}

	workdir := root
	log.Printf("[registry] preparing image in: %s", workdir)
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
			log.Println("[registry]", imageName, tags)
			if len(tags) != 1 {
				return "", fmt.Errorf("only one tag expected for %s", imageName)
			}
			for tag, hash := range tags {
				name = normalizeImageName(imageName)
				fmt.Printf("[registry] processing image:%s tag:%s hash:256:%s", name, tag, hash)
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

	configDest := fmt.Sprintf("%s/blobs/sha256:%s", workdir, string(configFile[:len(configFile)-5]))
	log.Printf("\n[registry] dist: %s", configDest)
	mkdir(configDest)
	if err := copyFile(tmp+"/"+configFile, configDest+"/"+configFile); err != nil {
		return "", err
	}

	mf, err := makeV2Manifest(manifest, configFile, configDest, tmp, workdir)
	if err != nil {
		return "", err
	}

	//spew.Dump(mf)

	err = writeJSON(mf, workdir+"/manifests/latest-v2")
	if err != nil {
		return "", err
	}

	return root, nil
}

func (registry *Registry) uploadDir(root string) (string, error) {
	hash, err := registry.ipfsClient.AddDir(root)
	if err != nil {
		return "", err
	}

	log.Printf("[registry] upload hash %s", hash)

	// get the first ref, which contains the image data
	refs, err := registry.ipfsClient.Refs(hash, false)
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

func copyio(out io.Reader, in io.Writer) error {
	_, err := io.Copy(in, out)
	if err != nil {
		return err
	}

	return nil
}

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
func makeV2Manifest(manifest map[string]interface{}, configFile, configDest, tmp, workdir string) (map[string]interface{}, error) {
	v2manifest, err := prepareV2Manifest(manifest, tmp, workdir+"/blobs")
	if err != nil {
		return nil, err
	}
	config := make(map[string]interface{})
	config["digest"] = "sha256:" + string(configFile[:len(configFile)-5])
	config["size"], err = fileSize(configDest + "/" + configFile)
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

func mergemap(a, b map[string]interface{}) map[string]interface{} {
	for k, v := range b {
		a[k] = v
	}
	return a
}

func prepareV2Manifest(mf map[string]interface{}, tmp, blobDir string) (map[string]interface{}, error) {
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
		size, digest, err := compressLayer(tmp+"/"+layer, blobDir)
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

func compressLayer(path, blobDir string) (int64, string, error) {
	log.Printf("[registry] compressing layer: %s", path)
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

func gzipFile(src, dst string) error {
	data, _ := ioutil.ReadFile(src)
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(data)
	w.Close()

	return ioutil.WriteFile(dst, b.Bytes(), os.ModePerm)
}

func fileSize(path string) (int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return int64(0), err
	}

	return fi.Size(), nil
}

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

func renameFile(src, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		return err
	}

	return nil
}

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

func readJSON(filepath string) (map[string]map[string]string, error) {
	body, _ := ioutil.ReadFile(filepath)
	var data map[string]map[string]string
	err := json.Unmarshal(body, &data)
	if err != nil {
		return data, err
	}

	return data, nil
}

func readJSONArray(filepath string) ([]map[string]interface{}, error) {
	body, _ := ioutil.ReadFile(filepath)
	var data []map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		return data, err
	}

	return data, nil
}

func normalizeImageName(name string) string {
	// TODO
	return name
}
