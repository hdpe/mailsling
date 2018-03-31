package mailer

import (
	"database/sql"
	"fmt"
)

type repositoryJournal struct {
	log  *Loggers
	repo Repository
}

func (j *repositoryJournal) SetRecipientPendingState(email string, lists []string, status RecipientStatus, attribs map[string]string) error {
	return j.repo.DoInTx(func(tx *sql.Tx) error {
		var recipientID int

		rec, found, err := j.repo.GetRecipientByEmail(tx, email)

		if err != nil {
			return fmt.Errorf("couldn't check for existing recipient: %v", err)
		} else if found {
			recipientID = rec.ID
		} else {
			recipientID, err = j.repo.InsertRecipient(tx, Recipient{Email: email})

			if err != nil {
				return fmt.Errorf("couldn't insert recipient: %v", err)
			}
		}

		for _, listID := range lists {
			var lr ListRecipient
			var lrFound bool
			if found {
				lr, lrFound, err = j.repo.GetListRecipientByEmailAndListID(tx, email, listID)
			}

			if err != nil {
				return fmt.Errorf("couldn't check for existing list recipient: %v", err)
			} else if lrFound {
				lr.status = status
				lr.attribs = attribs

				err = j.repo.UpdateListRecipient(tx, lr)

				if err != nil {
					return fmt.Errorf("couldn't update list recipient: %v", err)
				}
			} else {
				_, err = j.repo.InsertListRecipient(tx, ListRecipient{
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
	var result []listRecipientComposite
	var err error

	err = j.repo.DoInTx(func(tx *sql.Tx) error {
		var innerErr error
		result, innerErr = j.repo.GetRecipientDataByStatus(tx, []RecipientStatus{
			RecipientStatuses.Get("new"),
			RecipientStatuses.Get("unsubscribing")})

		return innerErr
	});

	return result, err;
}

func (j *repositoryJournal) UpdateListRecipient(listRecipientID int, status RecipientStatus) error {
	return j.repo.DoInTx(func(tx *sql.Tx) error {
		lr, err := j.repo.GetListRecipient(tx, listRecipientID)

		if err != nil {
			return fmt.Errorf("couldn't get existing list recipient: %v", err)
		}

		lr.status = status

		return j.repo.UpdateListRecipient(tx, lr)
	});
}

func newJournal(repo Repository) *repositoryJournal {
	return &repositoryJournal{repo: repo}
}
