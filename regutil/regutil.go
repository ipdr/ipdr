package regutil

import (
	"encoding/base32"
	"strings"

	base58 "github.com/jbenet/go-base58"
)

// DockerizeHash does base58 to base32 conversion
func DockerizeHash(base58Hash string) string {
	decodedB58 := base58.Decode(base58Hash)
	b32str := base32.StdEncoding.EncodeToString(decodedB58)
	// remove padding
	return strings.ToLower(b32str[0 : len(b32str)-1])
}

// IpfsifyHash does base32 to base58 conversion
func IpfsifyHash(base32Hash string) string {
	decodedB32, err := base32.StdEncoding.DecodeString(strings.ToUpper(base32Hash) + "=")
	if err != nil {
		return ""
	}

	return base58.Encode(decodedB32)
}
