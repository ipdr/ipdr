package registry

import (
	"sync"
)

// cids contains known cid entries.
type cids struct {
	// maps repo:tag -> cid
	cids map[string]string

	sync.RWMutex
}

func key(repo, ref string) string {
	return repo + ":" + ref
}

func (r *cids) add(repo, reference string, cid string) {
	r.Lock()

	if reference == "" {
		reference = "latest"
	}

	k := key(repo, reference)

	r.cids[k] = cid

	r.Unlock()
}

func (r *cids) get(repo, reference string) (string, bool) {
	r.RLock()

	if reference == "" {
		reference = "latest"
	}
	k := key(repo, reference)

	val, ok := r.cids[k]

	r.RUnlock()
	return val, ok
}
