package mailer

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestMailer_Poll(t *testing.T) {
	testCases := []struct {
		label         string
		defaultListID string

		getNextMessageResults []messageResult

		expectedSignUps []journalSignUp
		signUpResults   func(email string, lists []string) error

		expectedMessageSourceProcessed []Message

		expected string
	}{
		{
			label:         "on messages polled successfully",
			defaultListID: "a",

			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x"}`}},
				{msg: &testMessage{Text: `{"type":"sign_up","email":"y"}`}},
				{},
			},

			expectedSignUps: []journalSignUp{
				{email: "x", lists: []string{"a"}},
				{email: "y", lists: []string{"a"}},
			},

			expectedMessageSourceProcessed: []Message{
				&testMessage{Text: `{"type":"sign_up","email":"x"}`},
				&testMessage{Text: `{"type":"sign_up","email":"y"}`},
			},

			expected: "",
		},
		{
			label:         "on messages polled successfully with message list IDs specified",
			defaultListID: "",

			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x","listIds":["a","b"]}`}},
				{},
			},

			expectedSignUps: []journalSignUp{
				{email: "x", lists: []string{"a", "b"}},
			},

			expectedMessageSourceProcessed: []Message{
				&testMessage{Text: `{"type":"sign_up","email":"x","listIds":["a","b"]}`},
			},

			expected: "",
		},
		{
			label:         "on get next message error",
			defaultListID: "a",

			getNextMessageResults: []messageResult{
				{err: errors.New("")},
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x"}`}},
			},

			expectedSignUps: nil,

			expectedMessageSourceProcessed: nil,

			expected: "couldn't get next message",
		},
		{
			label:         "on couldn't parse sign up",
			defaultListID: "a",

			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: "{}"}},
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x"}`}},
				{},
			},

			expectedSignUps: []journalSignUp{
				{email: "x", lists: []string{"a"}},
			},

			expectedMessageSourceProcessed: []Message{&testMessage{Text: `{"type":"sign_up","email":"x"}`}},

			expected: "",
		},
		{
			label:         "on repository insert error",
			defaultListID: "a",

			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x"}`}},
				{msg: &testMessage{Text: `{"type":"sign_up","email":"y"}`}},
				{},
			},

			expectedSignUps: []journalSignUp{
				{email: "x", lists: []string{"a"}},
				{email: "y", lists: []string{"a"}},
			},
			signUpResults: func(email string, lists []string) error {
				if email == "x" {
					return errors.New("")
				}
				return nil
			},

			expectedMessageSourceProcessed: []Message{&testMessage{Text: `{"type":"sign_up","email":"y"}`}},

			expected: "",
		},
	}

	for _, tc := range testCases {
		ms := &testMessageSource{messageResults: tc.getNextMessageResults}
		j := &testJournal{signUpResults: tc.signUpResults}
		repo := struct{ Repository }{}

		mailer := &Mailer{log: NOOPLog, ms: ms, defaultlistID: tc.defaultListID, journal: j, repo: repo}

		err := mailer.Poll()

		if !reflect.DeepEqual(tc.expectedSignUps, j.signUpsReceived) {
			t.Errorf("%s expected to add to journal %v, actually %v", tc.label, tc.expectedSignUps, j.signUpsReceived)
		}
		if expected, actual := sliceVals(tc.expectedMessageSourceProcessed), sliceVals(ms.processed); !reflect.DeepEqual(expected, actual) {
			t.Errorf("%s expected message source to process %v, actually %v", tc.label, expected, actual)
		}
		if err != nil && tc.expected == "" {
			t.Errorf("%s didn't expect error but got %v", tc.label, err)
		} else if errorString := fmt.Sprintf("%v", err); strings.Index(errorString, tc.expected) != 0 {
			t.Errorf("%s expected error %v, actually %v", tc.label, tc.expected, err)
		}
	}
}

func sliceVals(msgs []Message) []string {
	res := make([]string, len(msgs))
	for i, m := range msgs {
		res[i] = fmt.Sprintf("%v", reflect.ValueOf(m).Elem())
	}
	return res
}

func TestMailer_Subscribe(t *testing.T) {
	testCases := []struct {
		label string

		repositoryGetNewRecipientsResult recipientsResult

		expectedClientReceived []subscription
		clientSubscribeResults map[subscription]error

		repositoryGetResults func(int) (ListRecipient, error)

		expectedRepositoryReceived []ListRecipient
		repositoryUpdateResults    map[ListRecipient]error

		expected string
	}{
		{
			label: "on repository recipients",

			repositoryGetNewRecipientsResult: recipientsResult{recipients: []listRecipientComposite{
				{recipientID: 1, email: "x", listID: "a"},
				{recipientID: 2, email: "y", listID: "b"},
			}},

			expectedClientReceived: []subscription{{email: "x", listID: "a"}, {email: "y", listID: "b"}},

			repositoryGetResults: func(id int) (ListRecipient, error) {
				return []ListRecipient{{id: 11}, {id: 12}}[id-1], nil
			},

			expectedRepositoryReceived: []ListRecipient{
				{id: 11, status: RecipientStatuses.Get("subscribed")},
				{id: 12, status: RecipientStatuses.Get("subscribed")},
			},

			expected: "",
		},
		{
			label: "on repository get new error",

			repositoryGetNewRecipientsResult: recipientsResult{err: errors.New("x")},

			expectedClientReceived: nil,

			expectedRepositoryReceived: nil,

			expected: "couldn't get recipients to be subscribed",
		},
		{
			label: "on client error",

			repositoryGetNewRecipientsResult: recipientsResult{recipients: []listRecipientComposite{{email: "x"}}},

			expectedClientReceived: []subscription{{email: "x"}},
			clientSubscribeResults: map[subscription]error{{email: "x"}: errors.New("")},

			repositoryGetResults: func(id int) (ListRecipient, error) {
				return ListRecipient{id: 1}, nil
			},

			expectedRepositoryReceived: []ListRecipient{
				{id: 1, status: RecipientStatuses.Get("failed")},
			},

			expected: "",
		},
		{
			label: "on repository get by id error",

			repositoryGetNewRecipientsResult: recipientsResult{recipients: []listRecipientComposite{{email: "x"}}},

			expectedClientReceived: []subscription{{email: "x"}},
			clientSubscribeResults: map[subscription]error{{email: "x"}: errors.New("")},

			repositoryGetResults: func(id int) (ListRecipient, error) {
				return ListRecipient{}, errors.New("")
			},

			expectedRepositoryReceived: nil,

			expected: "couldn't get recipient",
		},
		{
			label: "on repository update error",

			repositoryGetNewRecipientsResult: recipientsResult{recipients: []listRecipientComposite{{email: "x"}}},

			expectedClientReceived: []subscription{{email: "x"}},

			repositoryGetResults: func(id int) (ListRecipient, error) {
				return ListRecipient{id: 1}, nil
			},

			expectedRepositoryReceived: []ListRecipient{{id: 1, status: RecipientStatuses.Get("subscribed")}},
			repositoryUpdateResults: map[ListRecipient]error{
				ListRecipient{id: 1, status: RecipientStatuses.Get("subscribed")}: errors.New(""),
			},

			expected: "couldn't update recipient",
		},
	}

	for _, tc := range testCases {
		repo := &subscribeTestRepository{
			getNewRecipientsResult: tc.repositoryGetNewRecipientsResult,
			getRecipientResult:     tc.repositoryGetResults,
			updateRecipientResults: tc.repositoryUpdateResults,
		}
		client := &testClient{subscribeResults: tc.clientSubscribeResults}

		mailer := &Mailer{log: NOOPLog, repo: repo, client: client}

		err := mailer.Subscribe()

		if tc.expected != "" && (err == nil || strings.Index(fmt.Sprintf("%v", err), tc.expected) != 0) {
			t.Errorf("%s expected result %q, actually %q", tc.label, tc.expected, err)
		}
		if tc.expected == "" && err != nil {
			t.Errorf("%s expected nil result, actually %q", tc.label, err)
		}
		if !reflect.DeepEqual(tc.expectedClientReceived, client.received) {
			t.Errorf("%s expected client to receive %v, actually %v", tc.label, tc.expectedClientReceived, client.received)
		}
		if !reflect.DeepEqual(tc.expectedRepositoryReceived, repo.updateRecipientReceived) {
			t.Errorf("%s expected repository to receive %v, actually %v", tc.label, tc.expectedRepositoryReceived,
				repo.updateRecipientReceived)
		}
	}
}

func TestParseSignUp(t *testing.T) {
	testCases := []struct {
		label          string
		json           string
		expectedSignUp SignUpMessage
		expectedError  string
	}{
		{
			label:          "on valid json",
			json:           `{"type":"sign_up","email":"x"}`,
			expectedSignUp: SignUpMessage{Type: "sign_up", Email: "x"},
		},
		{
			label:         "on type not 'sign_up'",
			json:          `{"type":"_","email":"x@y.com"}`,
			expectedError: "message is not a sign up message",
		},
		{
			label:         "on no email",
			json:          `{"type":"sign_up"}`,
			expectedError: "message has no email",
		},
	}

	for _, tc := range testCases {
		signUp, err := parseSignUp(tc.json)

		if !reflect.DeepEqual(tc.expectedSignUp, signUp) {
			t.Errorf("%s expected %v, actually %v", tc.label, tc.expectedSignUp, signUp)
		}
		if errorString := fmt.Sprintf("%v", err); strings.Index(errorString, tc.expectedError) != 0 {
			t.Errorf("%s expected error %v, actually %v", tc.label, tc.expectedError, err)
		}
	}
}

// mocks

type messageResult struct {
	msg Message
	err error
}

type recipientResult struct {
	recipient Recipient
	found     bool
	err       error
}

type testMessageSource struct {
	idx            int
	messageResults []messageResult
	processed      []Message
}

func (ms *testMessageSource) GetNextMessage() (Message, error) {
	res := ms.messageResults[ms.idx]
	ms.idx++
	return res.msg, res.err
}

func (ms *testMessageSource) MessageProcessed(msg Message) error {
	ms.processed = append(ms.processed, msg)
	return nil
}

type testMessage struct {
	Text string
}

func (msg *testMessage) GetText() string {
	return msg.Text
}

type journalSignUp struct {
	email string
	lists []string
}

type testJournal struct {
	signUpsReceived []journalSignUp
	signUpResults   func(email string, lists []string) error
}

func (j *testJournal) SignUp(email string, lists []string) error {
	signUp := journalSignUp{email: email, lists: lists}
	j.signUpsReceived = append(j.signUpsReceived, signUp)
	if j.signUpResults == nil {
		return nil
	}
	return j.signUpResults(email, lists)
}

type insertRecipientResult struct {
	id  int
	err error
}

type recipientsResult struct {
	recipients []listRecipientComposite
	err        error
}

type subscribeTestRepository struct {
	Repository
	getNewRecipientsResult  recipientsResult
	getRecipientResult      func(id int) (ListRecipient, error)
	updateRecipientResults  map[ListRecipient]error
	updateRecipientReceived []ListRecipient
}

func (r *subscribeTestRepository) GetNewRecipients() ([]listRecipientComposite, error) {
	res := r.getNewRecipientsResult
	return res.recipients, res.err
}

func (r *subscribeTestRepository) GetRecipient(id int) (ListRecipient, error) {
	if r.getRecipientResult == nil {
		return ListRecipient{}, nil
	}
	return r.getRecipientResult(id)
}

func (r *subscribeTestRepository) UpdateRecipient(recipient ListRecipient) error {
	r.updateRecipientReceived = append(r.updateRecipientReceived, recipient)
	return r.updateRecipientResults[recipient]
}

type testClient struct {
	received         []subscription
	subscribeResults map[subscription]error
}

func (r *testClient) Subscribe(recipient subscription) error {
	r.received = append(r.received, recipient)
	return r.subscribeResults[recipient]
}
