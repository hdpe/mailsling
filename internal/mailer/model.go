package mailer

import "fmt"

type Recipient struct {
	ID    int
	Email string
}

type RecipientStatus string
type recipientStatusSet struct {
	None     RecipientStatus
	statuses []RecipientStatus
}

func (r recipientStatusSet) Get(name string) RecipientStatus {
	for _, us := range r.statuses {
		if string(us) == name {
			return us
		}
	}
	panic(fmt.Sprintf("Unknown status %q", name))
}

var RecipientStatuses = recipientStatusSet{
	statuses: []RecipientStatus{"new", "subscribed", "failed", "unsubscribing", "unsubscribed"},
	None:     RecipientStatus(""),
}

type ListRecipient struct {
	id          int
	listID      string
	recipientID int
	status      RecipientStatus
}
