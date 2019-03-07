package docker

import (
	"regexp"
	"strings"
)

// ShortImageID returns the short version of an image ID
func ShortImageID(imageID string) string {
	re := regexp.MustCompile(`(sha256:)?([0-9a-zA-Z]{12}).*`)
	return re.ReplaceAllString(imageID, `$2`)
}

// StripImageTagHost strips the host from an image tag
func StripImageTagHost(imageTag string) string {
	re := regexp.MustCompile(`(.*\..*?\/)?(.*)`)
	matches := re.FindStringSubmatch(imageTag)
	imageTag = matches[len(matches)-1]
	imageTag = strings.TrimPrefix(imageTag, "library/")
	return imageTag
}
