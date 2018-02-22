package main

import (
	"flag"
	"log"
	"os"

	"hdpe.me/remission-mailer/internal/mailer"
)

var (
	out = log.New(os.Stderr, "", 0)
)

func main() {
	var poll bool
	var subscribe bool

	flag.BoolVar(&poll, "poll", true, "poll SQS for new subscriptions")
	flag.BoolVar(&subscribe, "subscribe", true, "send subscription requests to MailChimp")
	flag.Parse()

	m := newMailer()

	if poll {
		err := m.Poll()

		if err != nil {
			out.Printf("Error polling for subscribe requests: %v", err)
		}
	}

	if subscribe {
		err := m.Subscribe()

		if err != nil {
			out.Printf("Error subscribing users: %v ", err)
		}
	}
}

func newMailer() *mailer.Mailer {
	ms, err := mailer.NewSQSMessageSource(os.Getenv("MAILER_SQS_URL"))

	if err != nil {
		out.Fatalf("Couldn't create SQS message source: %v", err)
	}

	repo, err := mailer.NewRepository(os.Getenv("MAILER_DB_DSN"))

	if err != nil {
		out.Fatalf("Couldn't create repository: %v", err)
	}

	client := mailer.NewClientConfig(
		os.Getenv("MAILER_MAILCHIMP_API_KEY"),
		os.Getenv("MAILER_MAILCHIMP_LIST_ID"),
	).NewClient()

	return mailer.NewMailer(ms, repo, client)
}
