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
		label                          string
		getNextMessageResults          []messageResult
		expectedRepositoryInserted     []User
		repositoryInsertResults        map[User]error
		expectedMessageSourceProcessed []Message
		expected                       string
	}{
		{
			label: "on messages polled successfully",
			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x"}`}},
				{msg: &testMessage{Text: `{"type":"sign_up","email":"y"}`}},
				{},
			},
			expectedRepositoryInserted: []User{
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
			expectedRepositoryInserted:     []User{{Email: "x"}},
			expectedMessageSourceProcessed: []Message{&testMessage{Text: `{"type":"sign_up","email":"x"}`}},
			expected:                       "",
		},
		{
			label: "on repository insert error",
			getNextMessageResults: []messageResult{
				{msg: &testMessage{Text: `{"type":"sign_up","email":"x"}`}},
				{msg: &testMessage{Text: `{"type":"sign_up","email":"y"}`}},
				{},
			},
			repositoryInsertResults: map[User]error{
				User{Email: "x"}: errors.New(""),
			},
			expectedRepositoryInserted: []User{
				{Email: "x"},
				{Email: "y"},
			},
			expectedMessageSourceProcessed: []Message{&testMessage{Text: `{"type":"sign_up","email":"y"}`}},
			expected:                       "",
		},
	}

	for _, tc := range testCases {
		ms := &testMessageSource{messageResults: tc.getNextMessageResults}
		repo := &pollTestRepository{insertResults: tc.repositoryInsertResults}

		mailer := &Mailer{ms: ms, repo: repo}

		err := mailer.Poll()

		if !reflect.DeepEqual(tc.expectedRepositoryInserted, repo.users) {
			t.Errorf("%s expected repo to insert %v, actually %v", tc.label, tc.expectedRepositoryInserted, repo.users)
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
		repositoryUsers            []User
		repositoryGetError         error
		clientError                error
		repositoryUpdateError      error
		expectedClientReceived     []User
		expectedRepositoryReceived []User
		expected                   string
	}{
		{
			label:                      "on repository users",
			repositoryUsers:            []User{{Email: "x"}},
			expectedClientReceived:     []User{{Email: "x"}},
			expectedRepositoryReceived: []User{{Email: "x", Status: UserStatuses.Get("subscribed")}},
			expected:                   "",
		},
		{
			label:                      "on repository get error",
			repositoryUsers:            []User{{}},
			repositoryGetError:         errors.New("x"),
			expectedClientReceived:     nil,
			expectedRepositoryReceived: nil,
			expected:                   "couldn't get users to be subscribed",
		},
		{
			label:                      "on client error",
			repositoryUsers:            []User{{}},
			clientError:                errors.New("x"),
			expectedClientReceived:     []User{{}},
			expectedRepositoryReceived: nil,
			expected:                   "notify of new user failed",
		},
		{
			label:                      "on repository update error",
			repositoryUsers:            []User{{}},
			repositoryUpdateError:      errors.New("x"),
			expectedClientReceived:     []User{{}},
			expectedRepositoryReceived: []User{{Status: UserStatuses.Get("subscribed")}},
			expected:                   "couldn't update user",
		},
	}

	for _, tc := range testCases {
		repo := &subscribeTestRepository{
			onGetUsersNotSubscribedUsers: tc.repositoryUsers,
			onGetUsersNotSubscribedError: tc.repositoryGetError,
			onUpdateUserError:            tc.repositoryUpdateError,
		}
		client := &testClient{shouldReturnError: tc.clientError}

		mailer := &Mailer{repo: repo, client: client}

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
		if !reflect.DeepEqual(tc.expectedRepositoryReceived, repo.updateUserReceived) {
			t.Errorf("%s expected repository to receive %v, actually %v", tc.label, tc.expectedRepositoryReceived,
				repo.updateUserReceived)
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
	insertResults map[User]error
	users         []User
}

func (r *pollTestRepository) GetUsersNotSubscribed() ([]User, error) {
	var result []User
	for _, u := range r.users {
		if u.Status == UserStatuses.Get("new") {
			result = append(result, u)
		}
	}
	return result, nil
}

func (r *pollTestRepository) InsertUser(user User) error {
	r.users = append(r.users, user)
	return r.insertResults[user]
}

func (r *pollTestRepository) UpdateUser(user User) error {
	for i, u := range r.users {
		if u.ID == user.ID {
			r.users[i] = user
			return nil
		}
	}
	panic(fmt.Sprintf("no such user %d", user.ID))
}

type subscribeTestRepository struct {
	Repository
	onGetUsersNotSubscribedUsers []User
	onGetUsersNotSubscribedError error
	updateUserReceived           []User
	onUpdateUserError            error
}

func (r *subscribeTestRepository) GetUsersNotSubscribed() ([]User, error) {
	if r.onGetUsersNotSubscribedError != nil {
		return nil, r.onGetUsersNotSubscribedError
	} else {
		return r.onGetUsersNotSubscribedUsers, nil
	}
}

func (r *subscribeTestRepository) UpdateUser(user User) error {
	r.updateUserReceived = append(r.updateUserReceived, user)
	return r.onUpdateUserError
}

type testClient struct {
	received          []User
	shouldReturnError error
}

func (r *testClient) SubscribeUser(signUp User) error {
	r.received = append(r.received, signUp)
	return r.shouldReturnError
}
