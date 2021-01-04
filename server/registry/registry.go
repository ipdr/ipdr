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
	"strconv"
	"strings"

	"github.com/miguelmota/ipdr/ipfs"
	"github.com/miguelmota/ipdr/regutil"
)

var contentTypes = map[string]string{
	"manifestV2Schema":     "application/vnd.docker.distribution.manifest.v2+json",
	"manifestListV2Schema": "application/vnd.docker.distribution.manifest.list.v2+json",
}

// Config is the config for the registry
type Config struct {
	IPFSHost     string
	IPFSGateway  string
	CIDResolvers []string
	CIDStorePath string
}

type registry struct {
	log       *log.Logger
	blobs     blobs
	manifests manifests

	cids *cidStore

	config     *Config
	ipfsClient *ipfs.Client

	resolver CIDResolver
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

func isDig(req *http.Request) bool {
	return req.URL.Path == "/dig/" || req.URL.Path == "/dig"
}

// dig resolves and returns cid or manifest content of the cid
// /dig?q=name:tag&short=true
func (r *registry) dig(resp http.ResponseWriter, req *http.Request) {
	parse := func(s string) bool {
		if b, err := strconv.ParseBool(s); err == nil {
			return b
		}
		return false
	}

	split := func(s string) (string, string) {
		sa := strings.SplitN(s, ":", 2)
		if len(sa) == 1 {
			return sa[0], ""
		}
		return sa[0], sa[1]
	}

	query := req.URL.Query()
	short := parse(query.Get("short"))
	name, tag := split(query.Get("q"))

	if name == "" {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(resp, "Required parameter 'q' missing. /dig?q=name:tag&short=true")
		return
	}

	list := r.resolve(name, tag)
	if len(list) == 0 {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	resp.WriteHeader(http.StatusOK)

	if tag != "" {
		cid := list[0]
		resp.Header().Set("X-Docker-Content-ID", cid)

		if short {
			fmt.Fprintln(resp, cid)
		} else {
			mf, err := r.manifests.getManifest(cid, tag)
			if err == nil {
				fmt.Fprintln(resp, string(mf.blob))
			}
		}
		return
	}

	// list
	for _, l := range list {
		fmt.Fprintln(resp, l)
	}
}

func (r *registry) root(resp http.ResponseWriter, req *http.Request) {
	if isDig(req) {
		r.dig(resp, req)
		return
	}
	if rerr := r.v2(resp, req); rerr != nil {
		r.log.Printf("%s %s %d %s %s", req.Method, req.URL, rerr.Status, rerr.Code, rerr.Message)
		rerr.Write(resp)
		return
	}
	r.log.Printf("%s %s", req.Method, req.URL)
}

// ipfsURL returns the full IPFS url
func (r *registry) ipfsURL(s []string) string {
	return regutil.IpfsURL(r.config.IPFSGateway, s)
}

// resolveCID returns content ID
// Lookup cid by repo:reference (tag/digest) via external services
// e.g. dnslink/ipns
func (r *registry) resolveCID(repo, reference string) (string, error) {
	if reference == "" {
		reference = "latest"
	}
	list := r.resolve(repo, reference)
	if len(list) > 0 {
		return list[0], nil
	}
	return "", fmt.Errorf("cannot resolve CID: %s:%s", repo, reference)
}

func (r *registry) resolve(repo, reference string) []string {
	// local/cached
	if cid, ok := r.cids.Get(repo, reference); ok {
		return []string{cid}
	}
	// repo is a valid cid, ignore reference and assume "latest"
	if cid := regutil.ToB32(repo); cid != "" {
		return []string{cid}
	}
	if hash := regutil.IpfsifyHash(repo); hash != "" {
		if cid := regutil.ToB32(hash); cid != "" {
			return []string{cid}
		}
	}

	// lookup
	return r.resolver.Resolve(repo, reference)
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
		cids:       newCIDStore(config.CIDStorePath),
		ipfsClient: ipfsClient,
		config:     config,
	}
	// TODO refactor so we donot have to do this?
	r.blobs.registry = r
	r.manifests.registry = r

	r.resolver = NewResolver(ipfsClient, config.CIDResolvers)

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
