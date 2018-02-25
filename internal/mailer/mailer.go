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
	log    *Loggers
	ms     MessageSource
	repo   Repository
	client Client
}

func (m *Mailer) Poll() error {
	for {
		msg, err := m.ms.GetNextMessage()
		if err != nil {
			return fmt.Errorf("couldn't get next message from queue: %v", err)
		} else if msg == nil {
			break
		}

		signUp, err := parseSignUp(msg.GetText())
		if err != nil {
			m.log.Error.Printf("couldn't parse sign up from message %q: %v", msg.GetText(), err)
			continue
		}

		_, found, err := m.repo.GetRecipientByEmail(signUp.Email)

		if err != nil {
			m.log.Error.Printf("couldn't get recipient by email from DB: %v", err)
			continue
		}

		if found {
			m.log.Error.Printf("recipient %q already known - skipping", signUp.Email)
		} else {
			err = m.repo.InsertRecipient(Recipient{Email: signUp.Email})
			if err != nil {
				m.log.Error.Printf("couldn't insert sign up to DB: %v", err)
				continue
			}
		}

		err = m.ms.MessageProcessed(msg)
		if err != nil {
			m.log.Error.Printf("couldn't mark message processed: %v", err)
		}
	}

	return nil
}

func (m *Mailer) Subscribe() error {
	rs, err := m.repo.GetNewRecipients()

	if err != nil {
		return fmt.Errorf("couldn't get recipients to be subscribed: %v", err)
	}

	for _, r := range rs {
		var status RecipientStatus

		err = m.client.Subscribe(r)

		if err != nil {
			m.log.Error.Printf("notify of new recipient failed: %v", err)
			status = RecipientStatuses.Get("failed")
		} else {
			status = RecipientStatuses.Get("subscribed")
		}

		r.Status = status

		err = m.repo.UpdateRecipient(r)
		if err != nil {
			return fmt.Errorf("couldn't update recipient: %v", err)
		}
	}

	return nil
}

func NewMailer(log *Loggers, ms MessageSource, repo Repository, client Client) *Mailer {
	return &Mailer{log, ms, repo, client}
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
