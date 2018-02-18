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

	ms, err := mailer.NewSQSMessageSource(os.Getenv("MAILER_SQS_URL"))

	if err != nil {
		log.Fatal("Error! ", err)
	}

	err = mailer.NewImporter(ms, repo).DoProcess()

	if err != nil {
		log.Fatal("Error! ", err)
	}
}
