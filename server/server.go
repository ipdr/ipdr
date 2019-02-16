package server

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	ipfs "github.com/miguelmota/ipdr/ipfs"
	regutil "github.com/miguelmota/ipdr/regutil"
	log "github.com/sirupsen/logrus"
)

// Server is server structure
type Server struct {
	debug    bool
	listener net.Listener
	host     string
}

// Config is server config
type Config struct {
	Debug bool
}

// NewServer returns a new server instance
func NewServer(config *Config) *Server {
	return &Server{
		host: fmt.Sprintf("0.0.0.0:%v", 5000),
	}
}

// Start runs the registry server
func (s *Server) Start() error {
	//  already listening
	if s.listener != nil {
		return nil
	}

	var gw string

	contentTypes := map[string]string{
		"manifestV2Schema":     "application/vnd.docker.distribution.manifest.v2+json",
		"manifestListV2Schema": "application/vnd.docker.distribution.manifest.list.v2+json",
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		uri := r.RequestURI
		s.Debugf("[registry/server] %s", uri)

		if uri == "/health" {
			fmt.Fprintln(w, "OK")
			return
		}

		if uri == "/v2/" {
			jsonstr := []byte(fmt.Sprintf(`{"what": "a registry", "gateway":%q, "handles": [%q, %q], "problematic": ["version 1 registries"], "project": "https://github.com/miguelmota/ipdr"}`, gw, contentTypes["manifestListV2Schema"], contentTypes["manifestV2Schema"]))

			w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
			fmt.Fprintln(w, string(jsonstr))
			return
		}

		if len(uri) <= 1 {
			fmt.Fprintln(w, "invalid multihash")
			return
		}

		var suffix string
		if strings.HasSuffix(uri, "/latest") {
			// docker daemon requesting the manifest
			suffix = "-v1"
			accepts := r.Header["Accept"]
			for _, accept := range accepts {
				if accept == contentTypes["manifestV2Schema"] ||
					accept == contentTypes["manifestListV2Schema"] {
					suffix = "-v2"
					break
				}
			}
		}

		parts := strings.Split(uri, "/")
		if len(parts) <= 2 {
			fmt.Fprintln(w, "out of range")
			return
		}

		hash := regutil.IpfsifyHash(parts[2])
		rest := strings.Join(parts[3:], "/") // tag
		path := hash + "/" + rest

		// blob request
		location := ipfsURL(path)

		if suffix != "" {
			// manifest request
			location = location + suffix
		}
		s.Debugf("[registry/server] location %s", location)

		req, err := http.NewRequest("GET", location, nil)
		if err != nil {
			fmt.Fprintf(w, err.Error())
			return
		}

		httpClient := http.Client{}
		resp, err := httpClient.Do(req)
		if err != nil {
			fmt.Fprintf(w, err.Error())
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(w, err.Error())
			return
		}

		//w.Header().Set("Location", location) // not required since we're fetching the content and proxying
		w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")

		// if latest-v2 set header
		w.Header().Set("Content-Type", contentTypes["manifestV2Schema"])
		fmt.Fprintf(w, string(body))
	})

	var err error
	s.listener, err = net.Listen("tcp", s.host)
	if err != nil {
		return err
	}

	port := s.listener.Addr().(*net.TCPAddr).Port
	s.Debugf("[registry/server] port %v", port)

	return http.Serve(s.listener, nil)
}

// Stop stops the server
func (s *Server) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
}

// Debugf prints debug log
func (s *Server) Debugf(str string, args ...interface{}) {
	if s.debug {
		log.Printf(str, args...)
	}
}

// ipfsURL returns the full IPFS url
func ipfsURL(hash string) string {
	url, err := ipfs.GatewayURL()
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%s/ipfs/%s", url, hash)
}
