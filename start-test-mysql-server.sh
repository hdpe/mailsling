#!/bin/sh
docker run \
    --rm \
    -e MYSQL_RANDOM_ROOT_PASSWORD=yes \
    -e MYSQL_DATABASE=mailer \
    -e MYSQL_USER=mailer \
    -e MYSQL_PASSWORD=password \
    -p 3306:3306 \
    mysql:5
