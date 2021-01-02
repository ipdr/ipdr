package image

// https://github.com/jvassev/image2ipfs/blob/master/client/manifest.go

// {
//     "schemaVersion": 2,
//     "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
//     "config": {
//         "mediaType":"application/vnd.docker.distribution.manifest.v2+json",
//         "size": 7023,
//         "digest": "sha256:b5b2b2c507a0944348e0303114d8d93aaaa081732b86451d9bce1f432a537bc7"
//     },
//     "layers": [
//         {
//             "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
//             "size": 32654,
//             "digest": "sha256:e692418e4cbaf90ca69d05a66403747baa33ee08806650b51fab815ad7fc331f"
//         }
//     ]
// }

const ManifestVersion = 2
const ManifestType = "application/vnd.docker.distribution.manifest.v2+json"
const ConfigType = "application/vnd.docker.container.image.v1+json"
const LayerType = "application/vnd.docker.image.rootfs.diff.tar.gzip"

type Config struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}
type Layer struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}
type Manifest struct {
	SchemaVersion int      `json:"schemaVersion"`
	MediaType     string   `json:"mediaType"`
	Config        *Config  `json:"config"`
	Layers        []*Layer `json:"layers"`
}

func (r *Manifest) Digests() []string {
	digests := []string{r.Config.Digest}
	for _, l := range r.Layers {
		digests = append(digests, l.Digest)
	}
	return digests
}
