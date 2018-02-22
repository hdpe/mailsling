package mailer

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestMailer_Poll(t *testing.T) {

	message0 := &testMessage{Text: `{"type":"sign_up","email":"x"}`}
	message1 := &testMessage{Text: `{"type":"sign_up","email":"y"}`}

	ms := &testMessageSource{messages: []Message{message0, message1}}
	repo := &pollTestRepository{}
	mailer := &Mailer{ms: ms, repo: repo}

	mailer.Poll()

	if expected, actual := 2, len(repo.users); expected != actual {
		t.Fatalf("Expected %d sign ups, was %d", expected, actual)
	}
	if expected, actual := "x", repo.users[0].Email; expected != actual {
		t.Errorf("Expected first persisted sign up email %s, was %s", expected, actual)
	}
	if processed := ms.processedMessages[message0]; !processed {
		t.Errorf("Expected first Message to be processed")
	}
	if expected, actual := "y", repo.users[1].Email; expected != actual {
		t.Errorf("Expected second persisted sign up email %s, was %s", expected, actual)
	}
	if processed := ms.processedMessages[message1]; !processed {
		t.Errorf("Expected second Message to be processed")
	}
}

func TestMailer_Subscribe(t *testing.T) {
	testCases := []struct {
		d                          string
		repositoryUsers            []User
		repositoryGetError         error
		clientError                error
		repositoryUpdateError      error
		expectedClientReceived     []User
		expectedRepositoryReceived []User
		expected                   string
	}{
		{
			d:                          "on repository users",
			repositoryUsers:            []User{{Email: "x"}},
			expectedClientReceived:     []User{{Email: "x"}},
			expectedRepositoryReceived: []User{{Email: "x", Status: UserStatuses.Get("welcomed")}},
			expected:                   "",
		},
		{
			d:                          "on repository get error",
			repositoryUsers:            []User{{}},
			repositoryGetError:         errors.New("x"),
			expectedClientReceived:     nil,
			expectedRepositoryReceived: nil,
			expected:                   "couldn't get users to be welcomed",
		},
		{
			d:                          "on client error",
			repositoryUsers:            []User{{}},
			clientError:                errors.New("x"),
			expectedClientReceived:     []User{{}},
			expectedRepositoryReceived: nil,
			expected:                   "notify of new user failed",
		},
		{
			d:                          "on repository update error",
			repositoryUsers:            []User{{}},
			repositoryUpdateError:      errors.New("x"),
			expectedClientReceived:     []User{{}},
			expectedRepositoryReceived: []User{{Status: UserStatuses.Get("welcomed")}},
			expected:                   "couldn't update user",
		},
	}

	for _, tc := range testCases {
		repo := &subscribeTestRepository{
			onGetUsersNotWelcomedUsers: tc.repositoryUsers,
			onGetUsersNotWelcomedError: tc.repositoryGetError,
			onUpdateUserError:          tc.repositoryUpdateError,
		}
		client := &testClient{shouldReturnError: tc.clientError}

		mailer := &Mailer{repo: repo, client: client}

		err := mailer.Subscribe()

		if tc.expected != "" && (err == nil || strings.Index(fmt.Sprintf("%v", err), tc.expected) != 0) {
			t.Errorf("%s expected result %q, actually %q", tc.d, tc.expected, err)
		}
		if tc.expected == "" && err != nil {
			t.Errorf("%s expected nil result, actually %q", tc.d, err)
		}
		if !reflect.DeepEqual(tc.expectedClientReceived, client.received) {
			t.Errorf("%s expected client to receive %v, actually %v", tc.d, tc.expectedClientReceived, client.received)
		}
		if !reflect.DeepEqual(tc.expectedRepositoryReceived, repo.updateUserReceived) {
			t.Errorf("%s expected repository to receive %v, actually %v", tc.d, tc.expectedRepositoryReceived,
				repo.updateUserReceived)
		}
	}
}

func TestParseSignUp(t *testing.T) {
	str := `{"type":"sign_up","email":"x@y.com"}`

	signUp, err := parseSignUp(str)

	if err != nil {
		t.Fatalf("error parsing json: %s", err)
	}

	if expected, actual := "sign_up", signUp.Type; expected != actual {
		t.Errorf("Expected %s, was %s", expected, actual)
	}
	if expected, actual := "x@y.com", signUp.Email; expected != actual {
		t.Errorf("Expected %s, was %s", expected, actual)
	}
}

// mocks

type testMessageSource struct {
	idx               int
	messages          []Message
	processedMessages map[Message]bool
}

func (ms *testMessageSource) GetNextMessage() (Message, error) {
	var msg Message
	if ms.idx < len(ms.messages) {
		msg = ms.messages[ms.idx]
		ms.idx++
	}
	return msg, nil
}

func (ms *testMessageSource) MessageProcessed(msg Message) error {
	if ms.processedMessages == nil {
		ms.processedMessages = map[Message]bool{}
	}
	ms.processedMessages[msg] = true
	return nil
}

type testMessage struct {
	Text string
}

func (msg *testMessage) GetText() string {
	return msg.Text
}

type pollTestRepository struct {
	users []*User
}

func (r *pollTestRepository) GetUsersNotWelcomed() ([]User, error) {
	var result []User
	for _, u := range r.users {
		if u.Status == UserStatuses.Get("new") {
			result = append(result, *u)
		}
	}
	return result, nil
}

func (r *pollTestRepository) InsertUser(user User) error {
	r.users = append(r.users, &user)
	return nil
}

func (r *pollTestRepository) UpdateUser(user User) error {
	for i, u := range r.users {
		if u.ID == user.ID {
			r.users[i] = &user
			return nil
		}
	}
	panic(fmt.Sprintf("no such user %d", user.ID))
}

type subscribeTestRepository struct {
	Repository
	onGetUsersNotWelcomedUsers []User
	onGetUsersNotWelcomedError error
	updateUserReceived         []User
	onUpdateUserError          error
}

func (r *subscribeTestRepository) GetUsersNotWelcomed() ([]User, error) {
	if r.onGetUsersNotWelcomedError != nil {
		return nil, r.onGetUsersNotWelcomedError
	} else {
		return r.onGetUsersNotWelcomedUsers, nil
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
