package mailer

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

type setRecipientStateMessage struct {
	Type    string   `json:"type"`
	Email   string   `json:"email"`
	ListIDs []string `json:"listIds"`
}

func (m setRecipientStateMessage) GetTargetStatus() (RecipientStatus, error) {
	switch {
	case m.Type == "sign_up" || m.Type == "subscribe":
		return RecipientStatuses.Get("new"), nil
	case m.Type == "unsubscribe":
		return RecipientStatuses.Get("unsubscribing"), nil
	}
	return RecipientStatuses.None, errors.New(fmt.Sprintf("unknown type: %v", m.Type))
}

type journal interface {
	SetRecipientPendingState(email string, lists []string, status RecipientStatus) error
	GetRecipientPendingState() ([]listRecipientComposite, error)
	UpdateListRecipient(listRecipientID int, status RecipientStatus) error
}

type notifier interface {
	Notify(s subscription, currentStatus RecipientStatus) (RecipientStatus, error)
}

type Mailer struct {
	log           *Loggers
	ms            MessageSource
	defaultlistID string
	journal       journal
	notifier      notifier
}

func (m *Mailer) Poll() error {
	for {
		msg, err := m.ms.GetNextMessage()
		if err != nil {
			return fmt.Errorf("couldn't get next message from queue: %v", err)
		} else if msg == nil {
			break
		}

		parsed, err := parseMessage(msg.GetText())
		if err != nil {
			m.log.Error.Printf("couldn't parse sign up from message %q: %v", msg.GetText(), err)
			continue
		}

		status, err := parsed.GetTargetStatus()
		if err != nil {
			m.log.Error.Printf("couldn't determine required status from message %q: %v", msg.GetText(), err)
			continue
		}

		err = m.journal.SetRecipientPendingState(parsed.Email, m.getListIDs(parsed), status)
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

func (m *Mailer) Process() error {
	rs, err := m.journal.GetRecipientPendingState()

	if err != nil {
		return fmt.Errorf("couldn't get recipients to be subscribed: %v", err)
	}

	for _, r := range rs {
		status, err := m.notifier.Notify(subscription{email: r.email, listID: r.listID}, r.status)

		if err != nil {
			m.log.Error.Printf("notify of new recipient failed: %v", err)
			status = RecipientStatuses.Get("failed")
		}

		err = m.journal.UpdateListRecipient(r.listRecipientID, status)
		if err != nil {
			return fmt.Errorf("couldn't update recipient: %v", err)
		}
	}

	return nil
}

func (m *Mailer) getListIDs(msg setRecipientStateMessage) []string {
	if len(msg.ListIDs) > 0 {
		return msg.ListIDs
	}
	return []string{m.defaultlistID}
}

func NewMailer(log *Loggers, ms MessageSource, listID string, repo Repository, client Client) *Mailer {
	return &Mailer{log, ms, listID,
		&repositoryJournal{log: log, repo: repo}, &clientNotifier{client: client}}
}

func parseMessage(str string) (msg setRecipientStateMessage, err error) {
	var parsed setRecipientStateMessage

	err = json.Unmarshal([]byte(str), &parsed)

	if err != nil {
		return
	}

	if parsed.Email == "" {
		err = fmt.Errorf("message has no email")
		return
	}

	return parsed, err
}
