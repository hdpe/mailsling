package mailer

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestRepositoryJournal_SetRecipientPendingState(t *testing.T) {
	testCases := []struct {
		label string

		email   string
		listIDs []string
		status  RecipientStatus

		getRecipientByEmailResult map[string]recipientResult

		expectedInsertRecipient Recipient
		insertRecipientInvoked  bool
		insertRecipientResult   insertRecipientResult

		getListRecipientByEmailAndListIDInvoked bool
		getListRecipientByEmailAndListIDResult  map[listRecipientByEmailAndListID]listRecipientResult

		expectedInsertListRecipients []ListRecipient
		insertListRecipientResult    map[ListRecipient]insertListRecipientResult

		expectedUpdateListRecipients []ListRecipient
		updateListRecipientResult    map[ListRecipient]error

		expectedAsString string
	}{
		{
			label:   "on new recipient",
			email:   "x",
			listIDs: []string{"a", "b"},
			status:  RecipientStatuses.Get("failed"),

			getRecipientByEmailResult: map[string]recipientResult{},

			expectedInsertRecipient: Recipient{Email: "x"},
			insertRecipientInvoked:  true,
			insertRecipientResult:   insertRecipientResult{id: 1},

			getListRecipientByEmailAndListIDInvoked: false,

			expectedInsertListRecipients: []ListRecipient{
				{recipientID: 1, listID: "a", status: RecipientStatuses.Get("failed")},
				{recipientID: 1, listID: "b", status: RecipientStatuses.Get("failed")},
			},

			expectedAsString: "",
		},
		{
			label:   "on existing recipient",
			email:   "x",
			listIDs: []string{"a"},
			status:  RecipientStatuses.Get("failed"),

			getRecipientByEmailResult: map[string]recipientResult{
				"x": {found: true, recipient: Recipient{ID: 1}},
			},

			insertRecipientInvoked: false,

			getListRecipientByEmailAndListIDInvoked: true,

			expectedInsertListRecipients: []ListRecipient{
				{recipientID: 1, listID: "a", status: RecipientStatuses.Get("failed")},
			},

			expectedAsString: "",
		},
		{
			label:   "on existing list recipient",
			email:   "x",
			listIDs: []string{"a"},
			status: RecipientStatuses.Get("unsubscribing"),

			getRecipientByEmailResult: map[string]recipientResult{
				"x": {found: true},
			},

			insertRecipientInvoked: false,

			getListRecipientByEmailAndListIDInvoked: true,
			getListRecipientByEmailAndListIDResult: map[listRecipientByEmailAndListID]listRecipientResult{
				listRecipientByEmailAndListID{email: "x", listID: "a"}: {found: true, listRecipient: ListRecipient{id: 1}},
			},

			expectedInsertListRecipients: nil,

			expectedUpdateListRecipients: []ListRecipient{{id: 1, status: RecipientStatuses.Get("unsubscribing")}},

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
			status:  RecipientStatuses.Get("failed"),

			getRecipientByEmailResult: map[string]recipientResult{
				"x": {found: true},
			},

			getListRecipientByEmailAndListIDInvoked: true,
			getListRecipientByEmailAndListIDResult:  nil,

			expectedInsertListRecipients: []ListRecipient{
				{listID: "a", status: RecipientStatuses.Get("failed")},
			},
			insertListRecipientResult: map[ListRecipient]insertListRecipientResult{
				ListRecipient{listID: "a", status: RecipientStatuses.Get("failed")}: {err: errors.New("")},
			},

			expectedAsString: "couldn't insert list recipient",
		},
		{
			label:   "on error on update list recipient",
			email:   "x",
			listIDs: []string{"a"},
			status:  RecipientStatuses.Get("unsubscribing"),

			getRecipientByEmailResult: map[string]recipientResult{
				"x": {found: true},
			},

			getListRecipientByEmailAndListIDInvoked: true,
			getListRecipientByEmailAndListIDResult: map[listRecipientByEmailAndListID]listRecipientResult{
				listRecipientByEmailAndListID{email: "x", listID: "a"}: {found: true, listRecipient: ListRecipient{id: 1}},
			},

			expectedUpdateListRecipients: []ListRecipient{{id: 1, status: RecipientStatuses.Get("unsubscribing")}},
			updateListRecipientResult: map[ListRecipient]error{
				ListRecipient{id: 1, status: RecipientStatuses.Get("unsubscribing")}: errors.New(""),
			},

			expectedAsString: "couldn't update list recipient",
		},
	}

	for _, tc := range testCases {
		r := &journalTestRepository{
			getRecipientByEmailResults:             tc.getRecipientByEmailResult,
			getListRecipientByEmailAndListIDResult: tc.getListRecipientByEmailAndListIDResult,
			insertRecipientResult:                  tc.insertRecipientResult,
			insertListRecipientResult:              tc.insertListRecipientResult,
			updateListRecipientResult:              tc.updateListRecipientResult,
		}

		j := &repositoryJournal{log: NOOPLog, repo: r}

		res := j.SetRecipientPendingState(tc.email, tc.listIDs, tc.status)

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
		if !reflect.DeepEqual(r.updateListRecipients, tc.expectedUpdateListRecipients) {
			t.Errorf("%s updated list recipients %v, expected %v", tc.label,
				r.updateListRecipients, tc.expectedUpdateListRecipients)
		}
		if tc.expectedAsString == "" && res != nil {
			t.Errorf("%s got error %v, none expected", tc.label, res)
		} else if resString := fmt.Sprintf("%v", res); strings.Index(resString, tc.expectedAsString) != 0 {
			t.Errorf("%s got error %v, expected %q", tc.label, res, tc.expectedAsString)
		}
	}
}

func TestRepositoryJournal_GetRecipientPendingState(t *testing.T) {
	testCases := []struct {
		label string

		onGetRecipientDataByStatus func([]RecipientStatus) ([]listRecipientComposite, error)

		expectedResult []listRecipientComposite
		expectedError  error
	}{
		{
			label: "gets new and unsubscribing recipients",

			onGetRecipientDataByStatus: func(statuses []RecipientStatus) ([]listRecipientComposite, error) {
				expected := []RecipientStatus{RecipientStatuses.Get("new"), RecipientStatuses.Get("unsubscribing")}
				if !reflect.DeepEqual(statuses, expected) {
					t.Errorf("gets new and unsubscribing recipients: GetRecipientDataByStatus "+
						"statuses want %v, got %v", statuses, expected)
				}
				return []listRecipientComposite{{email: "a@b.com"}}, nil
			},

			expectedResult: []listRecipientComposite{{email: "a@b.com"}},
			expectedError:  nil,
		},
		{
			label: "returns error on error",

			onGetRecipientDataByStatus: func(statuses []RecipientStatus) ([]listRecipientComposite, error) {
				return nil, errors.New("x")
			},

			expectedResult: nil,
			expectedError:  errors.New("x"),
		},
	}

	for _, tc := range testCases {
		r := &simpleTestRepository{
			onGetRecipientDataByStatus: tc.onGetRecipientDataByStatus,
		}
		j := &repositoryJournal{log: NOOPLog, repo: r}

		res, err := j.GetRecipientPendingState()

		if !r.getRecipientDataByStatusInvoked {
			t.Errorf("%v: invoked GetRecipientDataByStatus got %v, want %v", tc.label,
				r.getRecipientDataByStatusInvoked, true)
		}
		if !reflect.DeepEqual(res, tc.expectedResult) {
			t.Errorf("%v: result got %v, want %v", tc.expectedResult, res)
		}
		if !reflect.DeepEqual(err, tc.expectedError) {
			t.Errorf("%v: result got %v, want %v", tc.label, err, tc.expectedError)
		}
	}
}

func TestRepositoryJournal_UpdateListRecipient(t *testing.T) {
	testCases := []struct {
		label string

		listRecipientID int
		status          RecipientStatus

		onGetListRecipient         func(listRecipientID int) (ListRecipient, error)
		updateListRecipientInvoked bool
		onUpdateListRecipient      func(lr ListRecipient) error

		expected error
	}{
		{
			label: "updated with status",

			listRecipientID: 1,
			status:          RecipientStatuses.Get("subscribed"),

			onGetListRecipient: func(listRecipientID int) (ListRecipient, error) {
				if expected := 1; listRecipientID != expected {
					t.Errorf("updated with status: GetListRecipient got %v, want %v", listRecipientID, expected)
				}
				return ListRecipient{recipientID: 2}, nil
			},

			updateListRecipientInvoked: true,
			onUpdateListRecipient: func(lr ListRecipient) error {
				expected := ListRecipient{recipientID: 2, status: RecipientStatuses.Get("subscribed")}
				if lr != expected {
					t.Errorf("updated with status: UpdateListRecipient got %v, want %v", lr, expected)
				}
				return nil
			},

			expected: nil,
		},
		{
			label: "returns error on get list recipient error",

			onGetListRecipient: func(listRecipientID int) (ListRecipient, error) {
				return ListRecipient{}, errors.New("x")
			},

			updateListRecipientInvoked: false,

			expected: errors.New("couldn't get existing list recipient: x"),
		},
		{
			label: "returns error on update list recipient error",

			onGetListRecipient: func(listRecipientID int) (ListRecipient, error) {
				return ListRecipient{}, nil
			},

			updateListRecipientInvoked: true,
			onUpdateListRecipient: func(lr ListRecipient) error {
				return errors.New("x")
			},

			expected: errors.New("x"),
		},
	}

	for _, tc := range testCases {
		r := &simpleTestRepository{
			onGetListRecipient:    tc.onGetListRecipient,
			onUpdateListRecipient: tc.onUpdateListRecipient,
		}
		j := &repositoryJournal{log: NOOPLog, repo: r}

		err := j.UpdateListRecipient(tc.listRecipientID, tc.status)

		if !r.getListRecipientInvoked {
			t.Errorf("%v: GetListRecipient invoked got %v, want %v", tc.label, r.getListRecipientInvoked, true)
		}
		if r.updateListRecipientInvoked != tc.updateListRecipientInvoked {
			t.Errorf("%v: UpdateListRecipient invoked got %v, want %v", tc.label, r.updateListRecipientInvoked, tc.updateListRecipientInvoked)
		}
		if !reflect.DeepEqual(err, tc.expected) {
			t.Errorf("%v: got %q, want %q", tc.label, err, tc.expected)
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
	updateListRecipients                    []ListRecipient
	updateListRecipientResult               map[ListRecipient]error
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

func (r *journalTestRepository) UpdateListRecipient(lr ListRecipient) error {
	r.updateListRecipients = append(r.updateListRecipients, lr)
	return r.updateListRecipientResult[lr]
}

func (r *journalTestRepository) DoInTx(action func() error) error {
	return action()
}

type simpleTestRepository struct {
	Repository

	getRecipientDataByStatusInvoked bool
	onGetRecipientDataByStatus      func([]RecipientStatus) ([]listRecipientComposite, error)

	getListRecipientInvoked bool
	onGetListRecipient      func(listRecipientID int) (ListRecipient, error)

	updateListRecipientInvoked bool
	onUpdateListRecipient      func(listRecipient ListRecipient) error
}

func (r *simpleTestRepository) GetRecipientDataByStatus(statuses []RecipientStatus) ([]listRecipientComposite, error) {
	r.getRecipientDataByStatusInvoked = true
	return r.onGetRecipientDataByStatus(statuses)
}

func (r *simpleTestRepository) GetListRecipient(listRecipientID int) (ListRecipient, error) {
	r.getListRecipientInvoked = true
	return r.onGetListRecipient(listRecipientID)
}

func (r *simpleTestRepository) UpdateListRecipient(lr ListRecipient) error {
	r.updateListRecipientInvoked = true
	return r.onUpdateListRecipient(lr)
}
