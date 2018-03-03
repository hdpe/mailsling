package mailer

import (
	"encoding/json"
	"fmt"
)

type SignUpMessage struct {
	Type    string   `json:"type"`
	Email   string   `json:"email"`
	ListIDs []string `json:"listIds"`
}

type journal interface {
	SignUp(email string, lists []string) error
}

type Mailer struct {
	log           *Loggers
	ms            MessageSource
	defaultlistID string
	journal       journal
	repo          Repository
	client        Client
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

		err = m.journal.SignUp(signUp.Email, m.getListIDs(signUp))
		if err != nil {
			m.log.Error.Printf("%v", err)
			continue
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

		err = m.client.Subscribe(subscription{email: r.email, listID: r.listID})

		if err != nil {
			m.log.Error.Printf("notify of new recipient failed: %v", err)
			status = RecipientStatuses.Get("failed")
		} else {
			status = RecipientStatuses.Get("subscribed")
		}

		rec, err := m.repo.GetListRecipient(r.recipientID)
		if err != nil {
			return fmt.Errorf("couldn't get recipient: %v", err)
		}

		rec.status = status

		err = m.repo.UpdateListRecipient(rec)
		if err != nil {
			return fmt.Errorf("couldn't update recipient: %v", err)
		}
	}

	return nil
}

func (m *Mailer) getListIDs(signUp SignUpMessage) []string {
	if len(signUp.ListIDs) > 0 {
		return signUp.ListIDs
	}
	return []string{m.defaultlistID}
}

func NewMailer(log *Loggers, ms MessageSource, listID string, repo Repository, client Client) *Mailer {
	return &Mailer{log, ms, listID, &repositoryJournal{log: log, repo: repo}, repo, client}
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
