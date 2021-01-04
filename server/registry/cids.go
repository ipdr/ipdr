package registry

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// cidStore contains known cid entries.
type cidStore struct {
	// maps repo:tag -> cid
	cids     map[string]string
	location string

	sync.RWMutex
}

func key(repo, ref string) string {
	if ref == "" {
		ref = "latest"
	}
	return repo + ":" + ref
}

func (r *cidStore) Add(repo, reference string, cid string) {
	r.Lock()

	k := key(repo, reference)

	r.cids[k] = cid

	// only store name:tag reference
	if repo != cid && !strings.HasPrefix(reference, "sha256:") {
		r.writeCID(k, cid)
	}

	r.Unlock()
}

func (r *cidStore) Get(repo, reference string) (string, bool) {
	r.RLock()

	k := key(repo, reference)

	val, ok := r.cids[k]
	if !ok {
		if v, err := r.readCID(k); err == nil {
			val = v
			ok = true
		}
	}

	r.RUnlock()
	return val, ok
}

func (r *cidStore) readCID(key string) (string, error) {
	pc := strings.SplitN(key, ":", 2)
	p := filepath.Join(r.location, strings.Join(pc, "/"))
	content, err := ioutil.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (r *cidStore) writeCID(key string, val string) error {
	pc := strings.SplitN(key, ":", 2)
	p := filepath.Join(r.location, strings.Join(pc, "/"))
	if err := os.MkdirAll(filepath.Dir(p), os.ModePerm); err != nil {
		return err
	}

	return ioutil.WriteFile(p, []byte(val), 0644)
}

func newCIDStore(location string) *cidStore {

	return &cidStore{
		cids:     map[string]string{},
		location: location,
	}
}
