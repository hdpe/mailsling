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
		label                                string
		getNextMessageResults                []messageResult
		expectedRepositoryInserted           []Recipient
		repositoryGetRecipientByEmailResults map[string]recipientResult
		repositoryInsertResults              map[Recipient]error
		expectedMessageSourceProcessed       []Message
		expected                             string
	}{
		{
			label: "on messages polled successfully",
			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x"}`}},
				{msg: &testMessage{Text: `{"type":"sign_up","email":"y"}`}},
				{},
			},
			expectedRepositoryInserted: []Recipient{
				{Email: "x"},
				{Email: "y"},
			},
			expectedMessageSourceProcessed: []Message{
				&testMessage{Text: `{"type":"sign_up","email":"x"}`},
				&testMessage{Text: `{"type":"sign_up","email":"y"}`},
			},
			expected: "",
		},
		{
			label: "on get next message error",
			getNextMessageResults: []messageResult{
				{err: errors.New("")},
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x"}`}},
			},
			expectedRepositoryInserted:     nil,
			expectedMessageSourceProcessed: nil,
			expected:                       "couldn't get next message",
		},
		{
			label: "on couldn't parse sign up",
			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: "{}"}},
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x"}`}},
				{},
			},
			expectedRepositoryInserted:     []Recipient{{Email: "x"}},
			expectedMessageSourceProcessed: []Message{&testMessage{Text: `{"type":"sign_up","email":"x"}`}},
			expected:                       "",
		},
		{
			label: "on repository get returns recipient",
			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x"}`}},
				{},
			},
			repositoryGetRecipientByEmailResults: map[string]recipientResult{
				"x": {found: true},
			},
			expectedRepositoryInserted: nil,
			expectedMessageSourceProcessed: []Message{
				&testMessage{Text: `{"type":"sign_up","email":"x"}`},
			},
			expected: "",
		},
		{
			label: "on repository get error",
			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x"}`}},
				{msg: &testMessage{Text: `{"type":"sign_up","email":"y"}`}},
				{},
			},
			repositoryGetRecipientByEmailResults: map[string]recipientResult{
				"x": {err: errors.New("")},
			},
			expectedRepositoryInserted: []Recipient{
				{Email: "y"},
			},
			expectedMessageSourceProcessed: []Message{
				&testMessage{Text: `{"type":"sign_up","email":"y"}`},
			},
			expected: "",
		},
		{
			label: "on repository insert error",
			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x"}`}},
				{msg: &testMessage{Text: `{"type":"sign_up","email":"y"}`}},
				{},
			},
			repositoryInsertResults: map[Recipient]error{
				Recipient{Email: "x"}: errors.New(""),
			},
			expectedRepositoryInserted: []Recipient{
				{Email: "x"},
				{Email: "y"},
			},
			expectedMessageSourceProcessed: []Message{&testMessage{Text: `{"type":"sign_up","email":"y"}`}},
			expected:                       "",
		},
	}

	for _, tc := range testCases {
		ms := &testMessageSource{messageResults: tc.getNextMessageResults}
		repo := &pollTestRepository{
			getRecipientByEmailResults: tc.repositoryGetRecipientByEmailResults,
			insertResults:              tc.repositoryInsertResults,
		}

		mailer := &Mailer{log: NOOPLog, ms: ms, repo: repo}

		err := mailer.Poll()

		if !reflect.DeepEqual(tc.expectedRepositoryInserted, repo.recipients) {
			t.Errorf("%s expected repo to insert %v, actually %v", tc.label, tc.expectedRepositoryInserted, repo.recipients)
		}
		if !reflect.DeepEqual(tc.expectedMessageSourceProcessed, ms.processed) {
			t.Errorf("%s expected message source to process %v, actually %v", tc.label, tc.expectedMessageSourceProcessed,
				ms.processed)
		}
		if err != nil && tc.expected == "" {
			t.Errorf("%s didn't expect error but got %v", tc.label, err)
		} else if errorString := fmt.Sprintf("%v", err); strings.Index(errorString, tc.expected) != 0 {
			t.Errorf("%s expected error %v, actually %v", tc.label, tc.expected, err)
		}
	}
}

func TestMailer_Subscribe(t *testing.T) {
	testCases := []struct {
		label                      string
		repositoryRecipients       []Recipient
		repositoryGetError         error
		clientError                map[Recipient]error
		repositoryUpdateError      error
		expectedClientReceived     []Recipient
		expectedRepositoryReceived []Recipient
		expected                   string
	}{
		{
			label:                  "on repository recipients",
			repositoryRecipients:   []Recipient{{Email: "x"}, {Email: "y"}},
			expectedClientReceived: []Recipient{{Email: "x"}, {Email: "y"}},
			expectedRepositoryReceived: []Recipient{
				{Email: "x", Status: RecipientStatuses.Get("subscribed")},
				{Email: "y", Status: RecipientStatuses.Get("subscribed")},
			},
			expected: "",
		},
		{
			label:                      "on repository get error",
			repositoryGetError:         errors.New("x"),
			expectedClientReceived:     nil,
			expectedRepositoryReceived: nil,
			expected:                   "couldn't get recipients to be subscribed",
		},
		{
			label:                  "on client error",
			repositoryRecipients:   []Recipient{{Email: "x"}},
			clientError:            map[Recipient]error{{Email: "x"}: errors.New("")},
			expectedClientReceived: []Recipient{{Email: "x"}},
			expectedRepositoryReceived: []Recipient{
				{Email: "x", Status: RecipientStatuses.Get("failed")},
			},
			expected: "",
		},
		{
			label:                      "on repository update error",
			repositoryRecipients:       []Recipient{{}},
			repositoryUpdateError:      errors.New("x"),
			expectedClientReceived:     []Recipient{{}},
			expectedRepositoryReceived: []Recipient{{Status: RecipientStatuses.Get("subscribed")}},
			expected:                   "couldn't update recipient",
		},
	}

	for _, tc := range testCases {
		repo := &subscribeTestRepository{
			onGetRecipientsNotSubscribedRecipients: tc.repositoryRecipients,
			onGetRecipientsNotSubscribedError:      tc.repositoryGetError,
			onUpdateRecipientError:                 tc.repositoryUpdateError,
		}
		client := &testClient{subscribeRecipientResults: tc.clientError}

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

type pollTestRepository struct {
	Repository
	getRecipientByEmailResults map[string]recipientResult
	insertResults              map[Recipient]error
	recipients                 []Recipient
}

func (r *pollTestRepository) GetNewRecipients() ([]Recipient, error) {
	var result []Recipient
	for _, rec := range r.recipients {
		if rec.Status == RecipientStatuses.Get("new") {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (r *pollTestRepository) GetRecipientByEmail(email string) (Recipient, bool, error) {
	result := r.getRecipientByEmailResults[email]
	return result.recipient, result.found, result.err
}

func (r *pollTestRepository) InsertRecipient(recipient Recipient) error {
	r.recipients = append(r.recipients, recipient)
	return r.insertResults[recipient]
}

func (r *pollTestRepository) UpdateRecipient(recipient Recipient) error {
	for i, rec := range r.recipients {
		if rec.ID == recipient.ID {
			r.recipients[i] = recipient
			return nil
		}
	}
	panic(fmt.Sprintf("no such recipient %d", recipient.ID))
}

type subscribeTestRepository struct {
	Repository
	onGetRecipientsNotSubscribedRecipients []Recipient
	onGetRecipientsNotSubscribedError      error
	updateRecipientReceived                []Recipient
	onUpdateRecipientError                 error
}

func (r *subscribeTestRepository) GetNewRecipients() ([]Recipient, error) {
	if r.onGetRecipientsNotSubscribedError != nil {
		return nil, r.onGetRecipientsNotSubscribedError
	} else {
		return r.onGetRecipientsNotSubscribedRecipients, nil
	}
}

func (r *subscribeTestRepository) UpdateRecipient(recipient Recipient) error {
	r.updateRecipientReceived = append(r.updateRecipientReceived, recipient)
	return r.onUpdateRecipientError
}

type testClient struct {
	received                  []Recipient
	subscribeRecipientResults map[Recipient]error
}

func (r *testClient) Subscribe(recipient Recipient) error {
	r.received = append(r.received, recipient)
	return r.subscribeRecipientResults[recipient]
}
