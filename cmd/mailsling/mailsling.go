package main

import (
	"flag"
	"log"
	"os"

	"github.com/hdpe/mailsling/internal/mailer"
)

func main() {
	var poll bool
	var process bool

	flag.BoolVar(&poll, "poll", true, "poll SQS for new messages")
	flag.BoolVar(&process, "process", true, "notify clients of new recipient state")
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
	)

	client := mailer.NewClient(log, config)

	m := mailer.NewMailer(log, ms, os.Getenv("MAILER_MAILCHIMP_DEFAULT_LIST_ID"), repo, client)

	if poll {
		err := m.Poll()

		if err != nil {
			log.Error.Printf("Error polling for messages: %v", err)
		}
	}

	if process {
		err := m.Process()

		if err != nil {
			log.Error.Printf("Error processing recipient state: %v ", err)
		}
	}
}
