package main

import (
	"log"
	"os"

	"hdpe.me/remission-mailer/internal/mailer"
)

func main() {

	repo, err := mailer.NewRepository(os.Getenv("MAILER_DB_DSN"))

	if err != nil {
		log.Fatal("Error! ", err)
	}

	client := mailer.NewClientConfig(
		os.Getenv("MAILER_MAILCHIMP_DC"),
		os.Getenv("MAILER_MAILCHIMP_API_KEY"),
		os.Getenv("MAILER_MAILCHIMP_LIST_ID"),
	).NewClient()

	err = mailer.NewMailer(repo, client).ProcessOutstanding()

	if err != nil {
		log.Fatal("Error! ", err)
	}
}
