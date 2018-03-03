package mailer

import (
	"fmt"
	"time"
)

type Recipient struct {
	ID          int
	Email       string
	Status      RecipientStatus
	WelcomeTime time.Time
}

type RecipientStatus string
type recipientStatusSet []RecipientStatus

func (r recipientStatusSet) Get(name string) RecipientStatus {
	for _, us := range r {
		if string(us) == name {
			return us
		}
	}
	panic(fmt.Sprintf("Unknown status %q", name))
}

var RecipientStatuses = recipientStatusSet{"new", "subscribed", "failed"}

type ListRecipient struct {
	id          int
	listID      string
	recipientID int
	status      RecipientStatus
}
