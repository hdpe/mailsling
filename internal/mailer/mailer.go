package mailer

import (
	"encoding/json"
	"fmt"
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
			return fmt.Errorf("couldn't parse sign up from message %q: %v", msg.GetText(), err)
		}

		err = r.repo.InsertUser(User{Email: signUp.Email})
		if err != nil {
			return fmt.Errorf("couldn't insert sign up to DB: %v", err)
		}

		err = r.ms.MessageProcessed(msg)
		if err != nil {
			return fmt.Errorf("couldn't mark message processed: %v", err)
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

func parseSignUp(str string) (SignUpMessage, error) {
	var msg SignUpMessage

	err := json.Unmarshal([]byte(str), &msg)

	return msg, err
}
