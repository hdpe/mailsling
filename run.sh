#!/bin/sh
MAILER_SQS_URL=https://sqs.eu-west-2.amazonaws.com/330038105371/remission-sign-ups-dev \
    MAILER_DB_DSN='mailer:password@/mailer' \
    AWS_SDK_LOAD_CONFIG=1 \
    mailer
