package mailer

import (
	"encoding/json"
	"fmt"
	"log"
)

type SignUpMessage struct {
	Type  string `json:"type"`
	Email string `json:"email"`
}

type Mailer struct {
	ms     MessageSource
	repo   Repository
	client Client
}

func (r *Mailer) Poll() error {
	for {
		msg, err := r.ms.GetNextMessage()
		if err != nil {
			return fmt.Errorf("couldn't get next message from queue: %v", err)
		} else if msg == nil {
			break
		}

		signUp, err := parseSignUp(msg.GetText())
		if err != nil {
			log.Printf("couldn't parse sign up from message %q: %v", msg.GetText(), err)
			continue
		}

		err = r.repo.InsertUser(User{Email: signUp.Email})
		if err != nil {
			log.Printf("couldn't insert sign up to DB: %v", err)
			continue
		}

		err = r.ms.MessageProcessed(msg)
		if err != nil {
			log.Printf("couldn't mark message processed: %v", err)
		}
	}

	return nil
}

func (r *Mailer) Subscribe() error {
	users, err := r.repo.GetUsersNotWelcomed()

	if err != nil {
		return fmt.Errorf("couldn't get users to be welcomed: %v", err)
	}

	for _, u := range users {
		err = r.client.SubscribeUser(u)
		if err != nil {
			return fmt.Errorf("notify of new user failed: %v", err)
		}

		u.Status = "welcomed"

		err = r.repo.UpdateUser(u)
		if err != nil {
			return fmt.Errorf("couldn't update user: %v", err)
		}
	}

	return nil
}

func NewMailer(ms MessageSource, repo Repository, client Client) *Mailer {
	return &Mailer{ms, repo, client}
}

func parseSignUp(str string) (msg SignUpMessage, err error) {
	var parsed SignUpMessage

	err = json.Unmarshal([]byte(str), &parsed)

	if err != nil {
		return
	}

	if parsed.Type != "sign_up" {
		err = fmt.Errorf("message is not a sign up message")
		return
	}

	if parsed.Email == "" {
		err = fmt.Errorf("message has no email")
		return
	}

	return parsed, err
}
