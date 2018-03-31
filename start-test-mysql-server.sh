#!/bin/sh
docker run \
    --rm \
    --name mailsling_mysql \
    -e MYSQL_ROOT_PASSWORD=password \
    -e MYSQL_DATABASE=mailer \
    -e MYSQL_USER=mailer \
    -e MYSQL_PASSWORD=password \
    -p 3306:3306 \
    mysql:5
