#!/bin/bash

docker build -t example/helloworld .

docker run example/helloworld:latest

IPFS_HASH="$(ipdr push example/helloworld --silent)"

REPO_TAG=$(ipdr pull "$IPFS_HASH" --silent)

docker run "$REPO_TAG"

