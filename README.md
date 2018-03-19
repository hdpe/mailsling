# mailsling

[![Build Status](https://travis-ci.org/hdpe/mailsling.svg?branch=master)](https://travis-ci.org/hdpe/mailsling)

Baby's first golang program. Rigorously over-engineered; ruthlessly unidiomatic.

This program processes email sign-ups and unsubscribes, e.g. for newsletters, from a website or other thing. It:

* Rips messages out of an AWS SQS queue
* Parses recipient data from these
* De-dups recipients into a MySQL database, maintaining their subscription state here
* Subscribes/unsubscribes the recipients to/from one or more MailChimp lists

## Messages

This program expects messages of the following form:

```
{
    "type": "subscribe",
    "email": "ron@perlman.face",
    "listIds": ["12345abcde"]
}
```

or

```
{
    "type": "unsubscribe",
    "email": "ron@perlman.face"
}
```

## Configuration

All config via environment variables.

```ini
# AWS credentials/config:

AWS_ACCESS_KEY_ID=BLAHBLAHBLAH
AWS_SECRET_ACCESS_KEY=BlAhbLaHBLahBlAhbLaHBLahBlAhbLaHBLah
AWS_REGION=eu-west-2

# AWS SQS URL

MAILER_SQS_URL=https://sqs.eu-west-2.amazonaws.com/01234567890123/blah-queue

# MySQL go-sql-driver DSN - multiStatements parameter is required

MAILER_DB_DSN=mailer:password@/mailer?multiStatements=true

# MailChimp API key

MAILER_MAILCHIMP_API_KEY=BlAhbLaHBLahBlAhbLaHBLahBlAhbLaHBLah-us16

# MailChimp default list ID - optional, used if no lists specified in message

MAILER_MAILCHIMP_DEFAULT_LIST_ID=12345abcde

```

## Docker

The Docker image executes this program once a minute via crond.
