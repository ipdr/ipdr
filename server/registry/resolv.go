package registry

import (
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"sort"
	"strings"

	"github.com/miguelmota/ipdr/ipfs"
)

// CIDResolver is the interface that maps container image repo[:reference] to content ID.
type CIDResolver interface {
	Resolve(repo string, reference string) []string
}

// lookup resolves dnslink similar to the following
// https://github.com/ipfs/go-dnslink
func lookup(domain string) (string, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if !strings.HasPrefix(domain, "_dnslink.") {
		domain = "_dnslink." + domain
	}
	txts, err := net.LookupTXT(domain)
	if err != nil {
		return "", err
	}
	for _, txt := range txts {
		if txt != "" {
			if strings.HasPrefix(txt, "dnslink=") {
				txt = string(txt[8:])
			}
			return txt, nil
		}
	}
	return "", fmt.Errorf("invalid TXT record")
}

// File resolver
type fileResolver struct {
	root string
}

func NewFileResolver(uri string) (CIDResolver, error) {
	p := filepath.Clean(strings.TrimPrefix(uri, "file:"))
	return &fileResolver{
		root: p,
	}, nil
}

func (r *fileResolver) Resolve(repo, reference string) []string {
	if reference == "" {
		files, err := ioutil.ReadDir(fmt.Sprintf("%s/%s", r.root, repo))
		if err != nil {
			return nil
		}
		var sa []string
		for _, f := range files {
			if f.Mode().IsRegular() {
				sa = append(sa, f.Name())
			}
		}
		return sa
	}

	if b, err := ioutil.ReadFile(fmt.Sprintf("%s/%s/%s", r.root, repo, reference)); err == nil {
		return []string{strings.TrimSpace(string(b))}
	}
	return nil
}

// DNSLink resolver
// https://docs.ipfs.io/concepts/dnslink/
type dnslinkResolver struct {
	resolver CIDResolver
}

func NewDNSLinkResolver(client *ipfs.Client, domain string) (CIDResolver, error) {
	var r CIDResolver
	txt, err := lookup(domain)
	if err != nil {
		return nil, err
	}
	switch {
	case strings.HasPrefix(txt, "file:"):
		r, err = NewFileResolver(txt)
		if err != nil {
			return nil, err
		}
	case strings.HasPrefix(txt, "/ipfs/"):
		r, err = NewIPFSResolver(client, txt)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("not supported: %s", txt)
	}

	return &dnslinkResolver{
		resolver: r,
	}, nil
}

func (r *dnslinkResolver) Resolve(repo, reference string) []string {
	return r.resolver.Resolve(repo, reference)
}

// IPFS resolver
type ipfsResolver struct {
	client *ipfs.Client
	cid    string
}

func NewIPFSResolver(client *ipfs.Client, root string) (CIDResolver, error) {
	return &ipfsResolver{
		client: client,
		cid:    strings.TrimRight(strings.TrimPrefix(root, "/ipfs/"), "/"), // /ipfs/<cid>
	}, nil
}

func (r *ipfsResolver) Resolve(repo string, reference string) []string {
	if reference == "" {
		links, err := r.client.List(fmt.Sprintf("%s/%s", r.cid, repo))
		if err != nil {
			return nil
		}
		var sa []string
		for _, l := range links {
			sa = append(sa, l.Name)
		}
		return sa
	}

	if b, err := r.getContent(repo, reference); err == nil {
		return []string{strings.TrimSpace(string(b))}
	}
	return nil
}

func (r *ipfsResolver) getContent(repo, reference string) ([]byte, error) {
	rd, err := r.client.Cat(fmt.Sprintf("%s/%s/%s", r.cid, repo, reference))
	if err != nil {
		return nil, err
	}
	defer rd.Close()
	return ioutil.ReadAll(rd)
}

type resolver struct {
	resolvers []CIDResolver
}

func NewResolver(client *ipfs.Client, list []string) CIDResolver {
	var resolvers []CIDResolver
	for _, l := range list {
		switch {
		case strings.HasPrefix(l, "file:"):
			if r, err := NewFileResolver(l); err == nil {
				resolvers = append(resolvers, r)
			}
		case strings.HasPrefix(l, "/ipfs/"):
			if r, err := NewIPFSResolver(client, l); err == nil {
				resolvers = append(resolvers, r)
			}
		default:
			// assume dnslink
			if r, err := NewDNSLinkResolver(client, l); err == nil {
				resolvers = append(resolvers, r)
			}
		}
	}

	return &resolver{
		resolvers: resolvers,
	}
}

// collect all results if reference is empty for listing
func (r *resolver) Resolve(repo string, reference string) []string {
	var list []string
	for _, re := range r.resolvers {
		if result := re.Resolve(repo, reference); result != nil {
			// return early
			if reference != "" {
				return result
			}
			list = append(list, result...)
		}
	}
	list = uniq(list)
	sort.Strings(list)
	return list
}

func uniq(sa []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, s := range sa {
		if _, ok := keys[s]; !ok {
			keys[s] = true
			list = append(list, s)
		}
	}
	return list
}
