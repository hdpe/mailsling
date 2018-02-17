#!/bin/sh
docker run \
    --rm \
    -v "$(pwd)":/usr/src/hdpe.me/remission/mailer \
    -w /usr/src/hdpe.me/remission/mailer/mailer \
    -e GOPATH=/usr \
    -e GOOS=linux \
    golang:1.9.4 \
    go build -v