package main

import (
	"flag"
	"log"
	"os"

	"github.com/hdpe/mailsling/internal/mailer"
)

func main() {
	var poll bool
	var subscribe bool

	flag.BoolVar(&poll, "poll", true, "poll SQS for new subscriptions")
	flag.BoolVar(&subscribe, "subscribe", true, "send subscription requests to MailChimp")
	flag.Parse()

	log := &mailer.Loggers{
		Info:  log.New(os.Stdout, "", 0),
		Error: log.New(os.Stderr, "", 0),
	}

	ms, err := mailer.NewSQSMessageSource(log, os.Getenv("MAILER_SQS_URL"))

	if err != nil {
		log.Error.Fatalf("Couldn't create SQS message source: %v", err)
	}

	repo, err := mailer.NewRepository(os.Getenv("MAILER_DB_DSN"))

	if err != nil {
		log.Error.Fatalf("Couldn't create repository: %v", err)
	}

	defer repo.Close()

	config := mailer.NewClientConfig(
		os.Getenv("MAILER_MAILCHIMP_API_KEY"),
		os.Getenv("MAILER_MAILCHIMP_LIST_ID"),
	)

	client := mailer.NewClient(log, config)

	m := mailer.NewMailer(log, ms, repo, client)

	if poll {
		err := m.Poll()

		if err != nil {
			log.Error.Printf("Error polling for subscribe requests: %v", err)
		}
	}

	if subscribe {
		err := m.Subscribe()

		if err != nil {
			log.Error.Printf("Error subscribing users: %v ", err)
		}
	}
}
