package server

import (
	"fmt"
	"net"
	"net/http"

	ipfs "github.com/miguelmota/ipdr/ipfs"
	"github.com/miguelmota/ipdr/server/registry"
	log "github.com/sirupsen/logrus"
)

// Server is server structure
type Server struct {
	debug        bool
	listener     net.Listener
	host         string
	ipfsHost     string
	ipfsGateway  string
	cidResolvers []string
	cidStorePath string
	tlsCertPath  string
	tlsKeyPath   string
}

// Config is server config
type Config struct {
	Debug        bool
	Port         uint
	IPFSHost     string
	IPFSGateway  string
	CIDResolvers []string
	CIDStorePath string
	TLSCertPath  string
	TLSKeyPath   string
}

// InfoResponse is response for manifest info response
type InfoResponse struct {
	Info        string   `json:"what"`
	Project     string   `json:"project"`
	Gateway     string   `json:"gateway"`
	Handles     []string `json:"handles"`
	Problematic []string `json:"problematic"`
}

var projectURL = "https://github.com/miguelmota/ipdr"

// NewServer returns a new server instance
func NewServer(config *Config) *Server {
	if config == nil {
		config = &Config{}
	}

	var port uint = 5000
	if config.Port != 0 {
		port = config.Port
	}

	return &Server{
		host:         fmt.Sprintf("0.0.0.0:%v", port),
		debug:        config.Debug,
		ipfsHost:     config.IPFSHost,
		ipfsGateway:  ipfs.NormalizeGatewayURL(config.IPFSGateway),
		cidResolvers: config.CIDResolvers,
		cidStorePath: config.CIDStorePath,
		tlsCertPath:  config.TLSCertPath,
		tlsKeyPath:   config.TLSKeyPath,
	}
}

// Start runs the registry server
func (s *Server) Start() error {
	//  return if already started
	if s.listener != nil {
		return nil
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	http.Handle("/", registry.New(&registry.Config{
		IPFSHost:     s.ipfsHost,
		IPFSGateway:  s.ipfsGateway,
		CIDResolvers: s.cidResolvers,
		CIDStorePath: s.cidStorePath,
	}))

	var err error
	s.listener, err = net.Listen("tcp", s.host)
	if err != nil {
		return err
	}

	s.Debugf("[registry/server] listening on %s", s.listener.Addr())
	if s.tlsKeyPath != "" && s.tlsCertPath != "" {
		return http.ServeTLS(s.listener, nil, s.tlsCertPath, s.tlsKeyPath)
	}

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
