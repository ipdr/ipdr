package registry

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/miguelmota/ipdr/netutil"
	"github.com/miguelmota/ipdr/regutil"
)

func getContent(gw string, cid string, s []string) ([]byte, error) {
	uri := regutil.IpfsURL(gw, append([]string{cid}, s...))
	resp, err := netutil.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cid: %s %s", cid, resp.Status)
	}
	return ioutil.ReadAll(resp.Body)
}
