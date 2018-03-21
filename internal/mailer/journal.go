package mailer

import "fmt"

type repositoryJournal struct {
	log  *Loggers
	repo Repository
}

func (j *repositoryJournal) SetRecipientPendingState(email string, lists []string, status RecipientStatus, attribs map[string]string) error {
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
			var lr ListRecipient
			var lrFound bool
			if found {
				lr, lrFound, err = j.repo.GetListRecipientByEmailAndListID(email, listID)
			}

			if err != nil {
				return fmt.Errorf("couldn't check for existing list recipient: %v", err)
			} else if lrFound {
				lr.status = status
				lr.attribs = attribs

				err = j.repo.UpdateListRecipient(lr)

				if err != nil {
					return fmt.Errorf("couldn't update list recipient: %v", err)
				}
			} else {
				_, err = j.repo.InsertListRecipient(ListRecipient{
					recipientID: recipientID,
					listID:      listID,
					status:      status,
					attribs:     attribs,
				})

				if err != nil {
					return fmt.Errorf("couldn't insert list recipient: %v", err)
				}
			}
		}
		return nil
	})
}

func (j *repositoryJournal) GetRecipientPendingState() ([]listRecipientComposite, error) {
	return j.repo.GetRecipientDataByStatus([]RecipientStatus{
		RecipientStatuses.Get("new"),
		RecipientStatuses.Get("unsubscribing")})
}

func (j *repositoryJournal) UpdateListRecipient(listRecipientID int, status RecipientStatus) error {
	lr, err := j.repo.GetListRecipient(listRecipientID)

	if err != nil {
		return fmt.Errorf("couldn't get existing list recipient: %v", err)
	}

	lr.status = status

	return j.repo.DoInTx(func() error {
		return j.repo.UpdateListRecipient(lr)
	})
}

func newJournal(repo Repository) *repositoryJournal {
	return &repositoryJournal{repo: repo}
}
