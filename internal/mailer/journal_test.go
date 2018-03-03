package mailer

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestRepositoryJournal_SignUp(t *testing.T) {
	testCases := []struct {
		label   string
		email   string
		listIDs []string

		getRecipientByEmailResult map[string]recipientResult

		expectedInsertRecipient Recipient
		insertRecipientInvoked  bool
		insertRecipientResult   insertRecipientResult

		getListRecipientByEmailAndListIDInvoked bool
		getListRecipientByEmailAndListIDResult  map[listRecipientByEmailAndListID]listRecipientResult

		expectedInsertListRecipients []ListRecipient
		insertListRecipientResult    map[ListRecipient]insertListRecipientResult

		expectedAsString string
	}{
		{
			label:   "on new recipient",
			email:   "x",
			listIDs: []string{"a", "b"},

			getRecipientByEmailResult: map[string]recipientResult{},

			expectedInsertRecipient: Recipient{Email: "x"},
			insertRecipientInvoked:  true,
			insertRecipientResult:   insertRecipientResult{id: 1},

			getListRecipientByEmailAndListIDInvoked: false,

			expectedInsertListRecipients: []ListRecipient{
				{recipientID: 1, listID: "a", status: RecipientStatuses.Get("new")},
				{recipientID: 1, listID: "b", status: RecipientStatuses.Get("new")},
			},

			expectedAsString: "",
		},
		{
			label:   "on existing recipient",
			email:   "x",
			listIDs: []string{"a"},

			getRecipientByEmailResult: map[string]recipientResult{
				"x": {found: true, recipient: Recipient{ID: 1}},
			},

			insertRecipientInvoked: false,

			getListRecipientByEmailAndListIDInvoked: true,

			expectedInsertListRecipients: []ListRecipient{
				{recipientID: 1, listID: "a", status: RecipientStatuses.Get("new")},
			},

			expectedAsString: "",
		},
		{
			label:   "on existing list recipient",
			email:   "x",
			listIDs: []string{"a"},

			getRecipientByEmailResult: map[string]recipientResult{
				"x": {found: true},
			},

			insertRecipientInvoked: false,

			getListRecipientByEmailAndListIDInvoked: true,
			getListRecipientByEmailAndListIDResult: map[listRecipientByEmailAndListID]listRecipientResult{
				listRecipientByEmailAndListID{email: "x", listID: "a"}: {found: true},
			},

			expectedInsertListRecipients: nil,

			expectedAsString: "",
		},
		{
			label:   "on error on get recipient by email",
			email:   "x",
			listIDs: []string{"a"},

			getRecipientByEmailResult: map[string]recipientResult{
				"x": {err: errors.New("")},
			},

			insertRecipientInvoked: false,

			getListRecipientByEmailAndListIDInvoked: false,

			expectedInsertListRecipients: nil,

			expectedAsString: "couldn't check for existing recipient",
		},
		{
			label:   "on error on insert recipient",
			email:   "x",
			listIDs: []string{"a"},

			expectedInsertRecipient: Recipient{Email: "x"},
			insertRecipientInvoked:  true,
			insertRecipientResult:   insertRecipientResult{err: errors.New("")},

			expectedAsString: "couldn't insert recipient",
		},
		{
			label:   "on error on get list recipient by email and list ID",
			email:   "x",
			listIDs: []string{"a"},

			getRecipientByEmailResult: map[string]recipientResult{
				"x": {found: true},
			},

			getListRecipientByEmailAndListIDInvoked: true,
			getListRecipientByEmailAndListIDResult: map[listRecipientByEmailAndListID]listRecipientResult{
				listRecipientByEmailAndListID{email: "x", listID: "a"}: {err: errors.New("")},
			},

			expectedAsString: "couldn't check for existing list recipient",
		},
		{
			label:   "on error on insert list recipient",
			email:   "x",
			listIDs: []string{"a"},

			getRecipientByEmailResult: map[string]recipientResult{
				"x": {found: true},
			},

			getListRecipientByEmailAndListIDInvoked: true,
			getListRecipientByEmailAndListIDResult:  nil,

			expectedInsertListRecipients: []ListRecipient{
				{listID: "a", status: RecipientStatuses.Get("new")},
			},
			insertListRecipientResult: map[ListRecipient]insertListRecipientResult{
				ListRecipient{listID: "a", status: RecipientStatuses.Get("new")}: {err: errors.New("")},
			},

			expectedAsString: "couldn't insert list recipient",
		},
	}

	for _, tc := range testCases {
		r := &journalTestRepository{
			getRecipientByEmailResults:             tc.getRecipientByEmailResult,
			getListRecipientByEmailAndListIDResult: tc.getListRecipientByEmailAndListIDResult,
			insertRecipientResult:                  tc.insertRecipientResult,
			insertListRecipientResult:              tc.insertListRecipientResult,
		}

		j := &repositoryJournal{log: NOOPLog, repo: r}

		res := j.SignUp(tc.email, tc.listIDs)

		if r.insertRecipientInvoked != tc.insertRecipientInvoked {
			t.Errorf("%s invoked insert recipient = %v, expected %v", tc.label,
				r.insertRecipientInvoked, tc.insertRecipientInvoked)
		}
		if r.insertRecipient != tc.expectedInsertRecipient {
			t.Errorf("%s inserted recipient %v, expected %v", tc.label, r.insertRecipient, tc.expectedInsertRecipient)
		}
		if r.getListRecipientByEmailAndListIDInvoked != tc.getListRecipientByEmailAndListIDInvoked {
			t.Errorf("%s invoked get list recipient by email and list ID = %v, expected %v", tc.label,
				r.getListRecipientByEmailAndListIDInvoked, tc.getListRecipientByEmailAndListIDInvoked)
		}
		if !reflect.DeepEqual(r.insertListRecipients, tc.expectedInsertListRecipients) {
			t.Errorf("%s inserted list recipients %v, expected %v", tc.label,
				r.insertListRecipients, tc.expectedInsertListRecipients)
		}
		if tc.expectedAsString == "" && res != nil {
			t.Errorf("%s got error %v, none expected", tc.label, res)
		} else if resString := fmt.Sprintf("%v", res); strings.Index(resString, tc.expectedAsString) != 0 {
			t.Errorf("%s got error %v, expected %q", tc.label, res, tc.expectedAsString)
		}
	}
}

type listRecipientResult struct {
	listRecipient ListRecipient
	found         bool
	err           error
}

type listRecipientByEmailAndListID struct {
	email  string
	listID string
}

type insertListRecipientResult struct {
	id  int
	err error
}

type journalTestRepository struct {
	Repository
	getRecipientByEmailResults              map[string]recipientResult
	getListRecipientByEmailAndListIDResult  map[listRecipientByEmailAndListID]listRecipientResult
	getListRecipientByEmailAndListIDInvoked bool
	insertRecipient                         Recipient
	insertRecipientInvoked                  bool
	insertRecipientResult                   insertRecipientResult
	insertListRecipients                    []ListRecipient
	insertListRecipientResult               map[ListRecipient]insertListRecipientResult
}

func (r *journalTestRepository) GetRecipientByEmail(email string) (recipient Recipient, found bool, err error) {
	res := r.getRecipientByEmailResults[email]
	return res.recipient, res.found, res.err
}

func (r *journalTestRepository) InsertRecipient(rec Recipient) (int, error) {
	r.insertRecipient = rec
	r.insertRecipientInvoked = true
	res := r.insertRecipientResult
	return res.id, res.err
}

func (r *journalTestRepository) GetListRecipientByEmailAndListID(email string, listID string) (
	listRecipient ListRecipient, found bool, err error) {
	r.getListRecipientByEmailAndListIDInvoked = true
	res := r.getListRecipientByEmailAndListIDResult[listRecipientByEmailAndListID{email: email, listID: listID}]
	return res.listRecipient, res.found, res.err
}

func (r *journalTestRepository) InsertListRecipient(lr ListRecipient) (int, error) {
	r.insertListRecipients = append(r.insertListRecipients, lr)
	res := r.insertListRecipientResult[lr]
	return res.id, res.err
}

func (r *journalTestRepository) DoInTx(action func() error) error {
	return action()
}
