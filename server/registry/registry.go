// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package registry implements a docker V2 registry and the OCI distribution specification.
//
// It is designed to be used anywhere a low dependency container registry is needed, with an
// initial focus on tests.
//
// Its goal is to be standards compliant and its strictness will increase over time.
//
// This is currently a low flightmiles system. It's likely quite safe to use in tests; If you're using it
// in production, please let us know how and send us CL's for integration tests.
package registry

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	ipfs "github.com/miguelmota/ipdr/ipfs"
	"github.com/miguelmota/ipdr/regutil"
)

var contentTypes = map[string]string{
	"manifestV2Schema":     "application/vnd.docker.distribution.manifest.v2+json",
	"manifestListV2Schema": "application/vnd.docker.distribution.manifest.list.v2+json",
}

type registry struct {
	log       *log.Logger
	blobs     blobs
	manifests manifests

	cids cids

	config     *Config
	ipfsClient *ipfs.Client
}

// https://docs.docker.com/registry/spec/api/#api-version-check
// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#api-version-check
func (r *registry) v2(resp http.ResponseWriter, req *http.Request) *regError {
	if isBlob(req) {
		return r.blobs.handle(resp, req)
	}
	if isManifest(req) {
		return r.manifests.handle(resp, req)
	}
	resp.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	if req.URL.Path != "/v2/" && req.URL.Path != "/v2" {
		return &regError{
			Status:  http.StatusNotFound,
			Code:    "METHOD_UNKNOWN",
			Message: "We don't understand your method + url",
		}
	}
	resp.WriteHeader(200)
	return nil
}

func (r *registry) root(resp http.ResponseWriter, req *http.Request) {
	if rerr := r.v2(resp, req); rerr != nil {
		r.log.Printf("%s %s %d %s %s", req.Method, req.URL, rerr.Status, rerr.Code, rerr.Message)
		rerr.Write(resp)
		return
	}
	r.log.Printf("%s %s", req.Method, req.URL)
}

// ipfsURL returns the full IPFS url
func (r *registry) ipfsURL(s []string) string {
	return fmt.Sprintf("%s/ipfs/%s", r.config.IPFSGateway, strings.Join(s, "/"))
}

// resolveCID returns content ID
// TODO resolve cid by repo:reference (tag/digest) via external services
// e.g. dnslink/ipns
func (r *registry) resolveCID(repo, reference string) (string, error) {
	// local/cached
	if cid, ok := r.cids.get(repo, reference); ok {
		return cid, nil
	}
	// repo is a valid cid, ignore reference and assume "latest"
	if cid := regutil.ToB32(repo); cid != "" {
		return cid, nil
	}
	if cid := regutil.IpfsifyHash(repo); cid != "" {
		return regutil.ToB32(cid), nil
	}

	// TODO lookup cid by repo:reference
	return "", fmt.Errorf("cannot resolve CID: %s:%s", repo, reference)
}

// Config is the config for the registry
type Config struct {
	IPFSHost    string
	IPFSGateway string
}

// New returns a handler which implements the docker registry protocol.
// It should be registered at the site root.
func New(config *Config, opts ...Option) http.Handler {
	ipfsClient := ipfs.NewRemoteClient(&ipfs.Config{
		Host:       config.IPFSHost,
		GatewayURL: config.IPFSGateway,
	})
	r := &registry{
		log: log.New(os.Stderr, "", log.LstdFlags),
		blobs: blobs{
			contents: map[string][]byte{},
			uploads:  map[string][]byte{},
			layers:   map[string][]string{},
		},
		manifests: manifests{
			manifests: map[string]map[string]*manifest{},
		},
		cids: cids{
			cids: map[string]string{},
		},
		ipfsClient: ipfsClient,
		config:     config,
	}
	// TODO refactor so we donot have to do this?
	r.blobs.registry = r
	r.manifests.registry = r

	for _, o := range opts {
		o(r)
	}
	return http.HandlerFunc(r.root)
}

// Option describes the available options
// for creating the registry.
type Option func(r *registry)

// Logger overrides the logger used to record requests to the registry.
func Logger(l *log.Logger) Option {
	return func(r *registry) {
		r.log = l
	}
}
