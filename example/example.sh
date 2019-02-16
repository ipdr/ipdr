#!/bin/bash

# build Docker image
docker build -t example/helloworld .

# test run
docker run example/helloworld:latest

# push to IPFS
IPFS_HASH="$(ipdr push example/helloworld --silent)"

# pull from IPFS
REPO_TAG=$(ipdr pull "$IPFS_HASH" --silent)

# run image pulled from IPFS
docker run "$REPO_TAG"
