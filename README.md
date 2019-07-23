<h3 align="center">
  <br />
  <img src="https://user-images.githubusercontent.com/168240/52895983-7330f100-3176-11e9-855c-246eaabd3adc.png" alt="logo" width="600" />
  <br />
  <br />
  <br />
</h3>

# IPDR: InterPlanetary Docker Registry

> [IPFS](https://github.com/ipfs/go-ipfs)-backed [Docker](https://github.com/docker/docker) Registry

[![License](http://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/miguelmota/ipdr/master/LICENSE)
[![CircleCI](https://circleci.com/gh/miguelmota/ipdr.svg?style=svg)](https://circleci.com/gh/miguelmota/ipdr)
[![Go Report Card](https://goreportcard.com/badge/github.com/miguelmota/ipdr?)](https://goreportcard.com/report/github.com/miguelmota/ipdr)
[![GoDoc](https://godoc.org/github.com/miguelmota/ipdr?status.svg)](https://godoc.org/github.com/miguelmota/ipdr)
[![stability-experimental](https://img.shields.io/badge/stability-experimental-orange.svg)](https://github.com/emersion/stability-badges#experimental)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](#contributing)

IPDR is a [Docker Registry](https://docs.docker.com/registry/) tool that proxies Docker registry requests to IPFS for pushing and pulling images. IPDR allows you to store Docker images on IPFS instead of a central registry like Docker Hub or Google Container Registry. Docker images are referenced by their IPFS hash instead of the repo tag names.

IPDR is compatabile with the *Docker Registry HTTP [API V2 Spec](https://docs.docker.com/registry/spec/api/)* for pulling images&ast;

<sup><sub>&ast;not fully 1:1 implemented yet</sub></sup>

High-level overview:

<img src="https://user-images.githubusercontent.com/168240/52923314-14858780-32dc-11e9-80f8-9a0025de6090.png" alt="logo" width="500" />

## Contents

- [Install](#install)
- [Getting started](#getting-started)
- [CLI](#cli)
- [Test](#test)
- [FAQ](#faq)
- [Contributing](#contributing)
- [Resources](#resources)
- [License](#license)

## Install

- Install with [Go](https://golang.org/doc/install):

    ```bash
    go get -u github.com/miguelmota/ipdr/cmd/ipdr
    ```

- Install from [release binaries](https://github.com/miguelmota/ipdr/releases):

    ```bash
    # replace x.x.x with the latest version
    wget https://github.com/miguelmota/ipdr/releases/download/x.x.x/ipdr_x.x.x_linux_amd64.tar.gz
    tar -xvzf ipdr_x.x.x_linux_amd64.tar.gz ipdr
    ./ipdr --help

    # move to bin path
    sudo mv ipdr /usr/local/bin/ipdr
    ```

## Getting started

### Prerequisites

- Start IPFS daemon ([Install instructions](https://docs.ipfs.io/introduction/install/)):

    ```bash
    $ ipfs daemon
    Initializing daemon...
    Swarm listening on /ip4/127.0.0.1/tcp/4001
    Swarm listening on /ip4/192.168.86.90/tcp/4001
    Swarm listening on /ip6/::1/tcp/4001
    Swarm listening on /p2p-circuit/ipfs/QmR29wrbNv3WrMuodwuLiDwvskuZKKeTtcYDw7SwNffzCH
    Swarm announcing /ip4/127.0.0.1/tcp/4001
    Swarm announcing /ip4/192.168.0.21/tcp/43042
    Swarm announcing /ip4/192.168.86.90/tcp/4001
    Swarm announcing /ip6/::1/tcp/4001
    API server listening on /ip4/0.0.0.0/tcp/5001
    Gateway (readonly) server listening on /ip4/0.0.0.0/tcp/8080
    Daemon is ready
    ```

- Add `docker.localhost` to `/etc/hosts`:

    ```hosts
    echo '127.0.0.1 docker.localhost' | sudo tee -a /etc/hosts
    ```

    - Flush local DNS cache:

      - on macOS:

          ```bash
          dscacheutil -flushcache; sudo killall -HUP mDNSResponder
          ```

      - on Ubuntu 18+:

          ```bash
          sudo systemd-resolve --flush-caches
          ```

### Example flow

- Create `Dockerfile`:

    ```dockerfile
    FROM busybox:latest

    CMD echo 'hello world'
    ```

- Build Docker image:

    ```bash
    docker build -t example/helloworld .
    ```

    Test run:

    ```bash
    $ docker run example/helloworld:latest
    hello world
    ```

- Use IPDR CLI to push to IPFS:

    ```bash
    $ ipdr push example/helloworld

    INFO[0000] [registry] temp: /var/folders/k1/m2rmftgd48q97pj0xf9csdb00000gn/T/205139235
    INFO[0000] [registry] preparing image in: /var/folders/k1/m2rmftgd48q97pj0xf9csdb00000gn/T/657143846
    INFO[0000]
    [registry] dist: /var/folders/k1/m2rmftgd48q97pj0xf9csdb00000gn/T/657143846/default/blobs/sha256:305510b2c684403553fd8f383e8d109b147df2cfde60e40a85564532c383c8b8
    INFO[0000] [registry] compressing layer: /var/folders/k1/m2rmftgd48q97pj0xf9csdb00000gn/T/205139235/886f4bdfa483cc176e947c63d069579785c051793a9634f571fded7b9026cd3c/layer.tar
    INFO[0000] [registry] root dir: /var/folders/k1/m2rmftgd48q97pj0xf9csdb00000gn/T/657143846
    INFO[0000] [registry] upload hash QmRxZ5Wffj6b1j8ckJLcr7yFrbHUhBYXsAMbj7Krwu1pp8
    INFO[0000]
    [registry] uploaded to /ipfs/Qmc2ot2NQadXmbvPbsidyjYDvPfPwKZmovzNpfRPKxXUrL
    INFO[0000] [registry] docker image ciqmw4mig2uwaygddjlutoywq43udutvdmuxkcxvetsjp2mjdde27wi

    Successfully pushed Docker image to IPFS:
    /ipfs/Qmc2ot2NQadXmbvPbsidyjYDvPfPwKZmovzNpfRPKxXUrL
    ```

- Use IPDR CLI to pull from IPFS:

    ```bash
    $ ipdr pull /ipfs/QmagW4H1uE5rkm8A6iVS8WuiyjcWQzqXRHbM3KuUfzrCup

    INFO[0000] [registry/server] port 5000
    INFO[0000] [registry] attempting to pull docker.localhost:5000/ciqmw4mig2uwaygddjlutoywq43udutvdmuxkcxvetsjp2mjdde27wi
    INFO[0000] [registry/server] /v2/
    INFO[0000] [registry/server] /v2/ciqmw4mig2uwaygddjlutoywq43udutvdmuxkcxvetsjp2mjdde27wi/manifests/latest
    INFO[0000] [registry/server] location http://127.0.0.1:8080/ipfs/Qmc2ot2NQadXmbvPbsidyjYDvPfPwKZmovzNpfRPKxXUrL/manifests/latest-v2
    {"status":"Pulling from ciqmw4mig2uwaygddjlutoywq43udutvdmuxkcxvetsjp2mjdde27wi","id":"latest"}
    {"status":"Digest: sha256:1fb36e4704d6ebad5becdcfe996807de5f8db687da396330f112157c888c165b"}
    {"status":"Status: Downloaded newer image for docker.localhost:5000/ciqmw4mig2uwaygddjlutoywq43udutvdmuxkcxvetsjp2mjdde27wi:latest"}

    Successfully pulled Docker image from IPFS:
    docker.localhost:5000/ciqmw4mig2uwaygddjlutoywq43udutvdmuxkcxvetsjp2mjdde27wi
    ```

- Run image pulled from IPFS:

    ```bash
    $ docker run docker.localhost:5000/ciqmw4mig2uwaygddjlutoywq43udutvdmuxkcxvetsjp2mjdde27wi
    hello world
    ```

- Retag Docker image:

    ```bash
    $ docker tag docker.localhost:5000/ciqmw4mig2uwaygddjlutoywq43udutvdmuxkcxvetsjp2mjdde27wi example/helloworld:latest
    ```

- You can also pull the image using `docker pull`:
    - First run the IPDR server in a seperate terminal:

        ```bash
        $ ipdr server -p 5000
        INFO[0000] [registry/server] listening on [::]:5000
        ```

    - Then convert the IPFS hash to a valid format docker allows:

        ```bash
        $ ipdr convert QmYMg6WAuvF5i5yFmjT8KkqewZ5Ngh4U9Mp1bGfdjraFVk --format=docker

        ciqjjwaeoszdgcaasxmlhjuqnhbctgwijqz64w564lrzeyjezcvbj4y
        ```

    - Now you can `docker pull` the image from IPFS:

        ```bash
        $ docker pull docker.localhost:5000/ciqjjwaeoszdgcaasxmlhjuqnhbctgwijqz64w564lrzeyjezcvbj4y
        Using default tag: latest
        latest: Pulling from ciqjjwaeoszdgcaasxmlhjuqnhbctgwijqz64w564lrzeyjezcvbj4y
        Digest: sha256:6b787c9e04c2038d4b3cb0392417abdddfcfd88e10005d970fc751cdcfd6d895
        Status: Downloaded newer image for docker.localhost:5000/ciqjjwaeoszdgcaasxmlhjuqnhbctgwijqz64w564lrzeyjezcvbj4y:latest
        ```

        Test run:

        ```bash
        $ docker run docker.localhost:5000/ciqjjwaeoszdgcaasxmlhjuqnhbctgwijqz64w564lrzeyjezcvbj4y
        hello world
        ```

### TLDR; example

```bash
# build Docker image
docker build -t example/helloworld .

# push to IPFS
IPFS_HASH="$(ipdr push example/helloworld --silent)"

# pull from IPFS
REPO_TAG=$(ipdr pull "$IPFS_HASH" --silent)

# run image pulled from IPFS
docker run "$REPO_TAG"
```

## CLI

```bash
$ ipdr --help

The command-line interface for the InterPlanetary Docker Registry.
More info: https://github.com/miguelmota/ipdr

Usage:
  ipdr [flags]
  ipdr [command]

Available Commands:
  convert     Convert a hash to IPFS format or Docker registry format
  help        Help about any command
  pull        Pull image from the IPFS-backed Docker registry
  push        Push image to IPFS-backed Docker registry
  server      Start IPFS-backed Docker registry server

Flags:
  -h, --help   help for ipdr

Use "ipdr [command] --help" for more information about a command.
```

## Test

```bash
make test
```

## FAQ

- Q: How can I configure the local registry host or port that IPDR uses when pushing or pulling Docker images?

  - A: Use the `--docker-registry-host` flag, eg. `--docker-registry-host docker.for.mac.local:5000`

- Q: How can I configure the IPFS host that IPDR uses for pushing Docker images?

  - A: Use the `--ipfs-host` flag, eg. `--ipfs-host 127.0.0.1:5001`

- Q: How can I configure the IPFS gateway that IPDR uses for pulling Docker images?

  - A: Use the `--ipfs-gateway` flag, eg. `--ipfs-gateway https://ipfs.io`

- Q: How can I configure the port for the IPDR registry server?

  - A: Use the `--port` flag, eg. `--port 5000`

- Q: How can I use a HTTPS-secured registry when using `server`?

  - A: Use the `--tlsKeyPath` and `--tlsCrPath` flag, eg. ` --tlsKeyPath path/server.key --tlsCrtPath path/server.crt`

## Contributing

Pull requests are welcome!

For contributions please create a new branch and submit a pull request for review.

## Resources

- [Docker Registry HTTP API V2 Spec](https://docs.docker.com/registry/spec/api/)
- [Docker Registry 2.0 (slidedeck)](https://www.slideshare.net/Docker/docker-48351569)
- [image2ipfs](https://github.com/jvassev/image2ipfs/)

## License

[MIT](LICENSE)