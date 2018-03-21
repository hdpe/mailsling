package mailer

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestSetRecipientStateMessage_GetTargetStatus(t *testing.T) {
	testCases := []struct {
		label          string
		messageType    string
		expectedStatus RecipientStatus
		expectedError  error
	}{
		{
			label:          "type = 'sign_up'", //legacy
			messageType:    "sign_up",
			expectedStatus: RecipientStatuses.Get("new"),
			expectedError:  nil,
		},
		{
			label:          "type = 'subscribe'",
			messageType:    "subscribe",
			expectedStatus: RecipientStatuses.Get("new"),
			expectedError:  nil,
		},
		{
			label:          "type = 'unsubscribe'",
			messageType:    "unsubscribe",
			expectedStatus: RecipientStatuses.Get("unsubscribing"),
			expectedError:  nil,
		},
		{
			label:          "unknown type",
			messageType:    "x",
			expectedStatus: RecipientStatuses.None,
			expectedError:  errors.New("unknown type: x"),
		},
	}

	for _, tc := range testCases {
		msg := setRecipientStateMessage{Type: tc.messageType}

		status, err := msg.GetTargetStatus()

		if status != tc.expectedStatus {
			t.Errorf("%v: result status got %v, want %v", tc.label, status, tc.expectedStatus)
		}
		if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tc.expectedError) {
			t.Errorf("%v: result error got %q, want %q", tc.label, err, tc.expectedError)
		}
	}
}

func TestMailer_Poll(t *testing.T) {
	testCases := []struct {
		label         string
		defaultListID string

		getNextMessageResults []messageResult

		expectedPendingState []journalPendingState
		pendingStateResults  func(email string, lists []string) error

		expectedMessageSourceProcessed []Message

		expected string
	}{
		{
			label:         "on messages polled successfully",
			defaultListID: "a",

			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"unsubscribe","email":"x","attributes":{"k1":"v1"}}`}},
				{msg: &testMessage{Text: `{"type":"unsubscribe","email":"y","attributes":{"k2":"v2"}}`}},
				{},
			},

			expectedPendingState: []journalPendingState{
				{email: "x", lists: []string{"a"}, status: RecipientStatuses.Get("unsubscribing"), attribs: map[string]string{"k1": "v1"}},
				{email: "y", lists: []string{"a"}, status: RecipientStatuses.Get("unsubscribing"), attribs: map[string]string{"k2": "v2"}},
			},

			expectedMessageSourceProcessed: []Message{
				&testMessage{Text: `{"type":"unsubscribe","email":"x","attributes":{"k1":"v1"}}`},
				&testMessage{Text: `{"type":"unsubscribe","email":"y","attributes":{"k2":"v2"}}`},
			},

			expected: "",
		},
		{
			label:         "on messages polled successfully with message list IDs specified",
			defaultListID: "",

			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"subscribe","email":"x","listIds":["a","b"]}`}},
				{},
			},

			expectedPendingState: []journalPendingState{
				{email: "x", lists: []string{"a", "b"}, status: RecipientStatuses.Get("new")},
			},

			expectedMessageSourceProcessed: []Message{
				&testMessage{Text: `{"type":"subscribe","email":"x","listIds":["a","b"]}`},
			},

			expected: "",
		},
		{
			label:         "on get next message error",
			defaultListID: "a",

			getNextMessageResults: []messageResult{
				{err: errors.New("")},
				{msg: &testMessage{Text: `{"type":"subscribe","email":"x"}`}},
			},

			expectedPendingState: nil,

			expectedMessageSourceProcessed: nil,

			expected: "couldn't get next message",
		},
		{
			label:         "on couldn't parse sign up",
			defaultListID: "a",

			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: "!"}},
				{msg: &testMessage{Text: `{"type":"subscribe","email":"x"}`}},
				{},
			},

			expectedPendingState: []journalPendingState{
				{email: "x", lists: []string{"a"}, status: RecipientStatuses.Get("new")},
			},

			expectedMessageSourceProcessed: []Message{&testMessage{Text: `{"type":"subscribe","email":"x"}`}},

			expected: "",
		},
		{
			label:         "on couldn't determine required status",
			defaultListID: "a",

			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"_","email":"x"}`}},
				{msg: &testMessage{Text: `{"type":"subscribe","email":"y"}`}},
				{},
			},

			expectedPendingState: []journalPendingState{
				{email: "y", lists: []string{"a"}, status: RecipientStatuses.Get("new")},
			},

			expectedMessageSourceProcessed: []Message{&testMessage{Text: `{"type":"subscribe","email":"y"}`}},

			expected: "",
		},
		{
			label:         "on repository insert error",
			defaultListID: "a",

			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"subscribe","email":"x"}`}},
				{msg: &testMessage{Text: `{"type":"subscribe","email":"y"}`}},
				{},
			},

			expectedPendingState: []journalPendingState{
				{email: "x", lists: []string{"a"}, status: RecipientStatuses.Get("new")},
				{email: "y", lists: []string{"a"}, status: RecipientStatuses.Get("new")},
			},
			pendingStateResults: func(email string, lists []string) error {
				if email == "x" {
					return errors.New("")
				}
				return nil
			},

			expectedMessageSourceProcessed: []Message{&testMessage{Text: `{"type":"subscribe","email":"y"}`}},

			expected: "",
		},
	}

	for _, tc := range testCases {
		ms := &testMessageSource{messageResults: tc.getNextMessageResults}
		j := &testJournal{pendingStateResults: tc.pendingStateResults}

		mailer := &Mailer{log: NOOPLog, ms: ms, defaultlistID: tc.defaultListID, journal: j}

		err := mailer.Poll()

		if !reflect.DeepEqual(tc.expectedPendingState, j.pendingStateReceived) {
			t.Errorf("%v: invoked SetRecipientPendingState got %v, want %v", tc.label, j.pendingStateReceived, tc.expectedPendingState)
		}
		if actual, expected := sliceVals(ms.processed), sliceVals(tc.expectedMessageSourceProcessed); !reflect.DeepEqual(actual, expected) {
			t.Errorf("%v: invoked MessageProcessed got %v, got %v", tc.label, actual, expected)
		}
		if !errorMessageStartsWith(err, tc.expected) {
			t.Errorf("%v: result error got %q, want prefix %q", tc.label, err, tc.expected)
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

func TestMailer_Process(t *testing.T) {
	testCases := []struct {
		label string

		expectedNotifierReceived []notifyParams
		onNotify                 func(s subscription, currentStatus RecipientStatus) (RecipientStatus, error)

		onGetRecipientPendingState func() ([]listRecipientComposite, error)

		expectedUpdateListRecipientReceived []updateListRecipientParams
		onUpdateListRecipient               func(listRecipientID int, status RecipientStatus) error

		expected error
	}{
		{
			label: "on pending recipients",

			onGetRecipientPendingState: func() ([]listRecipientComposite, error) {
				return []listRecipientComposite{
					{listRecipientID: 1, email: "x", listID: "a", status: RecipientStatuses.Get("new")},
					{listRecipientID: 2, email: "y", listID: "b", status: RecipientStatuses.Get("new")},
				}, nil
			},

			expectedNotifierReceived: []notifyParams{
				{subscription: subscription{email: "x", listID: "a"}, currentStatus: RecipientStatuses.Get("new")},
				{subscription: subscription{email: "y", listID: "b"}, currentStatus: RecipientStatuses.Get("new")},
			},
			onNotify: func(s subscription, currentStatus RecipientStatus) (RecipientStatus, error) {
				return RecipientStatuses.Get("subscribed"), nil
			},

			expectedUpdateListRecipientReceived: []updateListRecipientParams{
				{listRecipientID: 1, status: RecipientStatuses.Get("subscribed")},
				{listRecipientID: 2, status: RecipientStatuses.Get("subscribed")},
			},
			onUpdateListRecipient: func(listRecipientID int, status RecipientStatus) error {
				return nil
			},

			expected: nil,
		},
		{
			label: "on get pending error",

			onGetRecipientPendingState: func() ([]listRecipientComposite, error) {
				return nil, errors.New("x")
			},

			expectedNotifierReceived: nil,

			expectedUpdateListRecipientReceived: nil,

			expected: errors.New("couldn't get recipients to be subscribed: x"),
		},
		{
			label: "on notifier error",

			onGetRecipientPendingState: func() ([]listRecipientComposite, error) {
				return []listRecipientComposite{
					{listRecipientID: 1, email: "x", listID: "a", status: RecipientStatuses.Get("new")},
				}, nil
			},

			expectedNotifierReceived: []notifyParams{
				{subscription: subscription{email: "x", listID: "a"}, currentStatus: RecipientStatuses.Get("new")},
			},
			onNotify: func(s subscription, currentStatus RecipientStatus) (RecipientStatus, error) {
				return RecipientStatuses.None, errors.New("")
			},

			expectedUpdateListRecipientReceived: []updateListRecipientParams{
				{listRecipientID: 1, status: RecipientStatuses.Get("failed")},
			},
			onUpdateListRecipient: func(listRecipientID int, status RecipientStatus) error {
				return nil
			},

			expected: nil,
		},
		{
			label: "on journal update error",

			onGetRecipientPendingState: func() ([]listRecipientComposite, error) {
				return []listRecipientComposite{
					{listRecipientID: 1, email: "x", listID: "a", status: RecipientStatuses.Get("new")},
				}, nil
			},

			expectedNotifierReceived: []notifyParams{
				{subscription: subscription{email: "x", listID: "a"}, currentStatus: RecipientStatuses.Get("new")},
			},
			onNotify: func(s subscription, currentStatus RecipientStatus) (RecipientStatus, error) {
				return RecipientStatuses.Get("subscribed"), nil
			},

			expectedUpdateListRecipientReceived: []updateListRecipientParams{
				{listRecipientID: 1, status: RecipientStatuses.Get("subscribed")},
			},
			onUpdateListRecipient: func(listRecipientID int, status RecipientStatus) error {
				return errors.New("x")
			},

			expected: errors.New("couldn't update recipient: x"),
		},
	}

	for _, tc := range testCases {
		j := &testJournal{
			onGetRecipientPendingState: tc.onGetRecipientPendingState,
			onUpdateListRecipient:      tc.onUpdateListRecipient,
		}
		notifier := &testClientNotifier{onNotify: tc.onNotify}

		mailer := &Mailer{log: NOOPLog, journal: j, notifier: notifier}

		err := mailer.Process()

		if !j.getRecipientPendingStateInvoked {
			t.Errorf("%v: invoked GetRecipientPendingState got %v, want %v", tc.label, j.getRecipientPendingStateInvoked, true)
		}
		if !reflect.DeepEqual(notifier.received, tc.expectedNotifierReceived) {
			t.Errorf("%v: invoked Notify got %v, want %v", tc.label, notifier.received, tc.expectedNotifierReceived)
		}
		if !reflect.DeepEqual(j.updateListRecipientReceived, tc.expectedUpdateListRecipientReceived) {
			t.Errorf("%v: invoked UpdateListRecipient params got %v, want %v", tc.label, j.updateListRecipientReceived, tc.expectedUpdateListRecipientReceived)
		}
		if !errorEquals(err, tc.expected) {
			t.Errorf("%v: result error got %q, want %q", tc.label, err, tc.expected)
		}
	}
}

func TestParseMessage(t *testing.T) {
	testCases := []struct {
		label           string
		json            string
		expectedMessage setRecipientStateMessage
		expectedError   string
	}{
		{
			label: "on valid json",
			json:  `{"type":"subscribe","email":"x","attributes":{"key":"value"}}`,
			expectedMessage: setRecipientStateMessage{
				Type:       "subscribe",
				Email:      "x",
				Attributes: map[string]string{"key": "value"},
			},
		},
		{
			label:         "on invalid json",
			json:          "{",
			expectedError: "invalid json: '{'",
		},
		{
			label:         "on no email",
			json:          `{"type":"sign_up"}`,
			expectedError: "message has no email",
		},
	}

	for _, tc := range testCases {
		msg, err := parseMessage(tc.json)

		if !reflect.DeepEqual(msg, tc.expectedMessage) {
			t.Errorf("%v: result got %v, want %v", tc.label, msg, tc.expectedMessage)
		}
		if !errorMessageStartsWith(err, tc.expectedError) {
			t.Errorf("%v: result error got %q, want prefix %q", tc.label, err, tc.expectedError)
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

type journalPendingState struct {
	email   string
	lists   []string
	status  RecipientStatus
	attribs map[string]string
}

type testJournal struct {
	journal
	pendingStateReceived []journalPendingState
	pendingStateResults  func(email string, lists []string) error

	getRecipientPendingStateInvoked bool
	onGetRecipientPendingState      func() ([]listRecipientComposite, error)

	updateListRecipientReceived []updateListRecipientParams
	onUpdateListRecipient       func(listRecipientID int, status RecipientStatus) error
}

func (j *testJournal) GetRecipientPendingState() ([]listRecipientComposite, error) {
	j.getRecipientPendingStateInvoked = true
	return j.onGetRecipientPendingState()
}

func (j *testJournal) UpdateListRecipient(listRecipientID int, status RecipientStatus) error {
	j.updateListRecipientReceived = append(j.updateListRecipientReceived, updateListRecipientParams{
		listRecipientID: listRecipientID,
		status:          status,
	})
	return j.onUpdateListRecipient(listRecipientID, status)
}

func (j *testJournal) SetRecipientPendingState(email string, lists []string, status RecipientStatus, attribs map[string]string) error {
	state := journalPendingState{email: email, lists: lists, status: status, attribs: attribs}
	j.pendingStateReceived = append(j.pendingStateReceived, state)
	if j.pendingStateResults == nil {
		return nil
	}
	return j.pendingStateResults(email, lists)
}

type notifyParams struct {
	subscription  subscription
	currentStatus RecipientStatus
}

type testClientNotifier struct {
	received []notifyParams
	onNotify func(s subscription, currentStatus RecipientStatus) (RecipientStatus, error)
}

func (n *testClientNotifier) Notify(s subscription, currentStatus RecipientStatus) (RecipientStatus, error) {
	n.received = append(n.received, notifyParams{subscription: s, currentStatus: currentStatus})
	return n.onNotify(s, currentStatus)
}

type updateListRecipientParams struct {
	listRecipientID int
	status          RecipientStatus
}
