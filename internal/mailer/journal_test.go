package mailer

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestRepositoryJournal_SetRecipientPendingState(t *testing.T) {
	testCases := []struct {
		label string

		email   string
		listIDs []string
		status  RecipientStatus
		attribs map[string]string

		time time.Time

		getRecipientByEmailResult map[string]recipientResult

		expectedInsertRecipient Recipient
		insertRecipientInvoked  bool
		onInsertRecipient       func(Recipient) (int, error)

		getListRecipientByEmailAndListIDInvoked bool
		onGetListRecipientByEmailAndListID      func(email string, listID string) (listRecipient ListRecipient, found bool, err error)

		expectedInsertListRecipients []ListRecipient
		onInsertListRecipient        func(ListRecipient) (int, error)

		expectedUpdateListRecipients []ListRecipient
		onUpdateListRecipient        func(ListRecipient) error

		expectedAsString string
	}{
		{
			label:   "on new recipient",
			email:   "x",
			listIDs: []string{"a", "b"},
			status:  RecipientStatuses.Get("new"),
			attribs: map[string]string{"k": "v"},

			time: time.Date(2018, 03, 28, 1, 2, 3, 4, time.Local),

			getRecipientByEmailResult: map[string]recipientResult{},

			expectedInsertRecipient: Recipient{Email: "x"},
			insertRecipientInvoked:  true,
			onInsertRecipient: func(recipient Recipient) (int, error) {
				return 1, nil
			},

			getListRecipientByEmailAndListIDInvoked: false,

			expectedInsertListRecipients: []ListRecipient{
				{recipientID: 1, listID: "a", status: RecipientStatuses.Get("new"), attribs: map[string]string{"k": "v"}, lastModified: time.Date(2018, 03, 28, 1, 2, 3, 4, time.Local)},
				{recipientID: 1, listID: "b", status: RecipientStatuses.Get("new"), attribs: map[string]string{"k": "v"}, lastModified: time.Date(2018, 03, 28, 1, 2, 3, 4, time.Local)},
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
			status:  RecipientStatuses.Get("unsubscribing"),

			time: time.Date(2018, 03, 28, 1, 2, 3, 4, time.Local),

			getRecipientByEmailResult: map[string]recipientResult{
				"x": {found: true},
			},

			insertRecipientInvoked: false,

			getListRecipientByEmailAndListIDInvoked: true,
			onGetListRecipientByEmailAndListID: func(email string, listID string) (listRecipient ListRecipient, found bool, err error) {
				return ListRecipient{id: 1}, true, nil
			},

			expectedInsertListRecipients: nil,

			expectedUpdateListRecipients: []ListRecipient{{id: 1, status: RecipientStatuses.Get("unsubscribing"), lastModified: time.Date(2018, 03, 28, 1, 2, 3, 4, time.Local)}},

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
			onInsertRecipient: func(recipient Recipient) (int, error) {
				return 0, errors.New("")
			},

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
			onGetListRecipientByEmailAndListID: func(email string, listID string) (listRecipient ListRecipient, found bool, err error) {
				return ListRecipient{}, false, errors.New("")
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
			onGetListRecipientByEmailAndListID: func(email string, listID string) (listRecipient ListRecipient, found bool, err error) {
				return ListRecipient{}, false, nil
			},

			expectedInsertListRecipients: []ListRecipient{
				{listID: "a", status: RecipientStatuses.Get("failed")},
			},
			onInsertListRecipient: func(recipient ListRecipient) (int, error) {
				return 0, errors.New("")
			},

			expectedAsString: "couldn't insert list recipient",
		},
		{
			label:   "on error on update list recipient",
			email:   "x",
			listIDs: []string{"a"},
			status:  RecipientStatuses.Get("unsubscribing"),
			attribs: map[string]string{"k": "v"},

			getRecipientByEmailResult: map[string]recipientResult{
				"x": {found: true},
			},

			getListRecipientByEmailAndListIDInvoked: true,
			onGetListRecipientByEmailAndListID: func(email string, listID string) (listRecipient ListRecipient, found bool, err error) {
				return ListRecipient{id: 1}, true, nil
			},

			expectedUpdateListRecipients: []ListRecipient{
				{
					id:      1,
					status:  RecipientStatuses.Get("unsubscribing"),
					attribs: map[string]string{"k": "v"},
				},
			},
			onUpdateListRecipient: func(recipient ListRecipient) error {
				return errors.New("")
			},

			expectedAsString: "couldn't update list recipient",
		},
	}

	for _, tc := range testCases {
		r := newJournalTestRepository(journalTestRepositoryParams{
			getRecipientByEmailResults:         tc.getRecipientByEmailResult,
			onGetListRecipientByEmailAndListID: tc.onGetListRecipientByEmailAndListID,
			onInsertRecipient:                  tc.onInsertRecipient,
			onInsertListRecipient:              tc.onInsertListRecipient,
			onUpdateListRecipient:              tc.onUpdateListRecipient,
		})

		j := &repositoryJournal{log: NOOPLog, repo: r, clock: &testClock{time: tc.time}}

		res := j.SetRecipientPendingState(tc.email, tc.listIDs, tc.status, tc.attribs)

		if r.insertRecipientInvoked != tc.insertRecipientInvoked {
			t.Errorf("%v: invoked InsertRecipient got %v, want %v", tc.label, r.insertRecipientInvoked, tc.insertRecipientInvoked)
		}
		if r.insertRecipient != tc.expectedInsertRecipient {
			t.Errorf("%v: invoked InsertRecipient params got %v, want %v", tc.label, r.insertRecipient, tc.expectedInsertRecipient)
		}
		if r.getListRecipientByEmailAndListIDInvoked != tc.getListRecipientByEmailAndListIDInvoked {
			t.Errorf("%v: invoked GetListRecipientByEmailAndListID got %v, want %v", tc.label,
				r.getListRecipientByEmailAndListIDInvoked, tc.getListRecipientByEmailAndListIDInvoked)
		}
		if !reflect.DeepEqual(r.insertListRecipients, tc.expectedInsertListRecipients) {
			t.Errorf("%v: invoked InsertListRecipient got %v, want %v", tc.label,
				r.insertListRecipients, tc.expectedInsertListRecipients)
		}
		if !reflect.DeepEqual(r.updateListRecipients, tc.expectedUpdateListRecipients) {
			t.Errorf("%v: invoked UpdateListRecipient got %v, want %v", tc.label,
				r.updateListRecipients, tc.expectedUpdateListRecipients)
		}
		if !errorMessageStartsWith(res, tc.expectedAsString) {
			t.Errorf("%v: result error got %q, want %q", tc.label, res, tc.expectedAsString)
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
			t.Errorf("%v: result got %v, want %v", tc.label, tc.expectedResult, res)
		}
		if !errorEquals(err, tc.expectedError) {
			t.Errorf("%v: result error got %v, want %v", tc.label, err, tc.expectedError)
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
				if !reflect.DeepEqual(lr, expected) {
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
		if !errorEquals(err, tc.expected) {
			t.Errorf("%v: result error got %q, want %q", tc.label, err, tc.expected)
		}
	}
}

type journalTestRepositoryParams struct {
	getRecipientByEmailResults map[string]recipientResult

	getListRecipientByEmailAndListIDInvoked bool
	onGetListRecipientByEmailAndListID      func(email string, listID string) (listRecipient ListRecipient, found bool, err error)

	insertRecipient        Recipient
	insertRecipientInvoked bool
	onInsertRecipient      func(Recipient) (int, error)

	insertListRecipients  []ListRecipient
	onInsertListRecipient func(ListRecipient) (int, error)

	updateListRecipients  []ListRecipient
	onUpdateListRecipient func(ListRecipient) error
}

type journalTestRepository struct {
	Repository
	journalTestRepositoryParams
}

func newJournalTestRepository(params journalTestRepositoryParams) *journalTestRepository {
	r := &journalTestRepository{journalTestRepositoryParams: params}

	if r.onGetListRecipientByEmailAndListID == nil {
		r.onGetListRecipientByEmailAndListID = func(email string, listID string) (listRecipient ListRecipient, found bool, err error) {
			return ListRecipient{}, false, nil
		}
	}
	if r.onInsertListRecipient == nil {
		r.onInsertListRecipient = func(recipient ListRecipient) (int, error) {
			return 0, nil
		}
	}
	if r.onUpdateListRecipient == nil {
		r.onUpdateListRecipient = func(recipient ListRecipient) error {
			return nil
		}
	}

	return r
}

func (r *journalTestRepository) GetRecipientByEmail(tx *sql.Tx, email string) (recipient Recipient, found bool, err error) {
	res := r.getRecipientByEmailResults[email]
	return res.recipient, res.found, res.err
}

func (r *journalTestRepository) InsertRecipient(tx *sql.Tx, rec Recipient) (int, error) {
	r.insertRecipient = rec
	r.insertRecipientInvoked = true
	return r.onInsertRecipient(rec)
}

func (r *journalTestRepository) GetListRecipientByEmailAndListID(tx *sql.Tx, email string, listID string) (
	listRecipient ListRecipient, found bool, err error) {
	r.getListRecipientByEmailAndListIDInvoked = true
	return r.onGetListRecipientByEmailAndListID(email, listID)
}

func (r *journalTestRepository) InsertListRecipient(tx *sql.Tx, lr ListRecipient) (int, error) {
	r.insertListRecipients = append(r.insertListRecipients, lr)
	return r.onInsertListRecipient(lr)
}

func (r *journalTestRepository) UpdateListRecipient(tx *sql.Tx, lr ListRecipient) error {
	r.updateListRecipients = append(r.updateListRecipients, lr)
	return r.onUpdateListRecipient(lr)
}

func (r *journalTestRepository) DoInTx(action func(*sql.Tx) error) error {
	return action(nil)
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

func (r *simpleTestRepository) GetRecipientDataByStatus(tx *sql.Tx, statuses []RecipientStatus) ([]listRecipientComposite, error) {
	r.getRecipientDataByStatusInvoked = true
	return r.onGetRecipientDataByStatus(statuses)
}

func (r *simpleTestRepository) GetListRecipient(tx *sql.Tx, listRecipientID int) (ListRecipient, error) {
	r.getListRecipientInvoked = true
	return r.onGetListRecipient(listRecipientID)
}

func (r *simpleTestRepository) UpdateListRecipient(tx *sql.Tx, lr ListRecipient) error {
	r.updateListRecipientInvoked = true
	return r.onUpdateListRecipient(lr)
}

func (r *simpleTestRepository) DoInTx(action func(*sql.Tx) error) error {
	return action(nil)
}

type testClock struct {
	time time.Time
}

func (c *testClock) now() time.Time {
	return c.time
}
