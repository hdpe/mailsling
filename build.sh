#!/bin/sh
docker run \
    --rm \
    -v "$(pwd)":/usr/src/app \
    -w /usr/src/app/mailer \
    -e GOOS=linux \
    golang:1.9.4 \
    go build -v