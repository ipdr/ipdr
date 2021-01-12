#!/bin/bash

# set -x
set -e
set -o pipefail

#
export IPFS_PATH=${IPFS_PATH:-$HOME/.ipdr/ipfs/data}

# build

if ! command -v ipdr &> /dev/null
then
    echo "  *** ipdr not found, building it..."
    (cd ../../../ipdr && go install ./cmd/ipdr)
fi

#
function build_run {
    # build Docker image
    docker build --quiet -t $1 --build-arg REPO_NAME=$1 .

    # test run
    docker run $1
}

function cleanup {
    docker rmi -f $(docker image ls -q $1)
}

#
my=docker.local:5000

###
repo_name="hello:v0.0.1-b$RANDOM"
echo "  *** push/pull $repo_name using ipdr..."
build_run $repo_name

# push to IPFS
IPFS_HASH="$(ipdr push $repo_name --silent)"

# pull from IPFS
REPO_TAG=$(ipdr pull "$IPFS_HASH" --silent)

# run image pulled from IPFS
docker run "$REPO_TAG"

# clean up
cleanup $repo_name

###
repo_name="hello:v0.0.1-b$RANDOM"
echo "  *** push/pull $repo_name using docker cli..."
build_run $repo_name

# push to IPFS
docker tag $repo_name $my/$repo_name
docker push --quiet $my/$repo_name

# pull from IPFS
docker pull --quiet $my/$repo_name

# run image pulled from IPFS
docker run $my/$repo_name

# clean up
cleanup $repo_name

###
echo "  compatibility tests..."

###
repo_name="hello:v0.0.1-b$RANDOM"
echo "  *** ipdr push/docker pull $repo_name..."
build_run $repo_name

# push to IPFS
IPFS_HASH="$(ipdr push $repo_name --silent)"

# pull from IPFS
docker pull --quiet $my/$IPFS_HASH


# run image pulled from IPFS
docker run $my/$IPFS_HASH

# clean up
cleanup $repo_name

###
repo_name="hello:v0.0.1-b$RANDOM"
echo "  *** docker push/ipdr pull $repo_name..."
build_run $repo_name

# push to IPFS
docker tag $repo_name $my/$repo_name
docker push --quiet $my/$repo_name

# pull from IPFS
IPFS_HASH=$(ipdr dig $repo_name --short=true)
REPO_TAG=$(ipdr pull "$IPFS_HASH" --silent)

# run image pulled from IPFS
docker run "$REPO_TAG"

# clean up
cleanup $repo_name
echo "test complete."
