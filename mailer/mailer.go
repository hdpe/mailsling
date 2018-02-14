package main

import (
	"log"
	"os"
	"hdpe.me/remission/mailer/internal/mailer"
)

func main() {

	persister, err := mailer.NewPersister(os.Getenv("MAILER_DB_DSN"))

	if err != nil {
		log.Fatal("Error! ", err)
	}

	ms, err := mailer.NewSQSMessageSource(os.Getenv("MAILER_SQS_URL"))

	if err != nil {
		log.Fatal("Error! ", err)
	}

	err = mailer.DoProcess(ms, persister)

	if err != nil {
		log.Fatal("Error! ", err)
	}
}