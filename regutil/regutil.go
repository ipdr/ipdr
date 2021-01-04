package regutil

import (
	"encoding/base32"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	cid "github.com/ipfs/go-cid"
	base58 "github.com/jbenet/go-base58"
	mbase "github.com/multiformats/go-multibase"
)

// DockerizeHash does base58 to base32 conversion
func DockerizeHash(base58Hash string) string {
	re := regexp.MustCompile(`(/ipfs/)?(.*)`)
	matches := re.FindStringSubmatch(base58Hash)
	base58Hash = matches[len(matches)-1]
	decodedB58 := base58.Decode(base58Hash)
	b32str := base32.StdEncoding.EncodeToString(decodedB58)

	end := len(b32str)
	if end > 0 {
		end = end - 1
	}

	// remove padding
	return strings.ToLower(b32str[0:end])
}

// IpfsifyHash does base32 to base58 conversion
func IpfsifyHash(base32Hash string) string {
	decodedB32, err := base32.StdEncoding.DecodeString(strings.ToUpper(base32Hash) + "=")
	if err != nil {
		return ""
	}

	return base58.Encode(decodedB32)
}

// IpfsURL constructs IPFS gateway URL
func IpfsURL(gw string, s []string) string {
	return fmt.Sprintf("%s/ipfs/%s", gw, strings.Join(s, "/"))
}

func toCidV0(c cid.Cid) (cid.Cid, error) {
	if c.Type() != cid.DagProtobuf {
		return cid.Cid{}, fmt.Errorf("can't convert non-protobuf nodes to cidv0")
	}
	return cid.NewCidV0(c.Hash()), nil
}

func toCidV1(c cid.Cid) (cid.Cid, error) {
	return cid.NewCidV1(c.Type(), c.Hash()), nil
}

// ToB58 returns base58 encoded string if s is a valid cid
func ToB58(s string) string {
	c, err := cid.Decode(s)
	if err == nil {
		return c.Hash().B58String()
	}
	return ""
}

// ToB32 returns base32 encoded string if s is a valid cid
func ToB32(s string) string {
	c, err := cid.Decode(s)
	if err != nil {
		return ""
	}
	c1, err := toCidV1(c)
	if err != nil {
		return ""
	}
	b32, err := c1.StringOfBase(mbase.Base32)
	if err != nil {
		return ""
	}
	return b32
}

// Dig interrogates registry server. It performs CID lookups and shows the response.
func Dig(gw string, short bool, name string) (string, error) {
	uri := fmt.Sprintf("http://%s/dig?q=%s&short=%v", gw, name, short)

	resp, err := http.Get(uri)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(resp.Status)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
