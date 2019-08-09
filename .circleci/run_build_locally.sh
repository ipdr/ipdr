#!/usr/bin/env bash

curl --user "${CIRCLE_TOKEN}:" \
    --request POST \
     --form revision=0a042d26a7bdb34291c175d8603dbe8bfb21ad7b\
    --form config=@config.yml \
    --form notify=false \
        https://circleci.com/api/v1.1/project/github/miguelmota/ipdr/tree/master
