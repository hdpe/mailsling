package mailer

import "fmt"

type repositoryJournal struct {
	log  *Loggers
	repo Repository
}

func (j *repositoryJournal) SignUp(email string, lists []string) error {
	return j.repo.DoInTx(func() error {
		var recipientID int

		rec, found, err := j.repo.GetRecipientByEmail(email)

		if err != nil {
			return fmt.Errorf("couldn't check for existing recipient: %v", err)
		} else if found {
			recipientID = rec.ID
		} else {
			recipientID, err = j.repo.InsertRecipient(Recipient{Email: email})

			if err != nil {
				return fmt.Errorf("couldn't insert recipient: %v", err)
			}
		}

		for _, listID := range lists {
			var lrFound bool
			if found {
				_, lrFound, err = j.repo.GetListRecipientByEmailAndListID(email, listID)
			}

			if err != nil {
				return fmt.Errorf("couldn't check for existing list recipient: %v", err)
			} else if lrFound {
				j.log.Error.Printf("recipient %s already known to list %s: skipping", email, listID)
			} else {
				_, err = j.repo.InsertListRecipient(ListRecipient{
					recipientID: recipientID,
					listID:      listID,
					status:      RecipientStatuses.Get("new"),
				})

				if err != nil {
					return fmt.Errorf("couldn't insert list recipient: %v", err)
				}
			}
		}
		return nil
	})
}

func newJournal(repo Repository) *repositoryJournal {
	return &repositoryJournal{repo: repo}
}
