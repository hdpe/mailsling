package mailer

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestMailer_ProcessOutstanding(t *testing.T) {
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
		repo := &mailerTestRepository{
			onGetUsersNotWelcomedUsers: tc.repositoryUsers,
			onGetUsersNotWelcomedError: tc.repositoryGetError,
			onUpdateUserError:          tc.repositoryUpdateError,
		}
		client := &testClient{shouldReturnError: tc.clientError}

		mailer := &Mailer{repo: repo, client: client}

		err := mailer.ProcessOutstanding()

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

type testClient struct {
	received          []User
	shouldReturnError error
}

func (r *testClient) SubscribeUser(signUp User) error {
	r.received = append(r.received, signUp)
	return r.shouldReturnError
}

type mailerTestRepository struct {
	Repository
	onGetUsersNotWelcomedUsers []User
	onGetUsersNotWelcomedError error
	updateUserReceived         []User
	onUpdateUserError          error
}

func (r *mailerTestRepository) GetUsersNotWelcomed() ([]User, error) {
	if r.onGetUsersNotWelcomedError != nil {
		return nil, r.onGetUsersNotWelcomedError
	} else {
		return r.onGetUsersNotWelcomedUsers, nil
	}
}

func (r *mailerTestRepository) UpdateUser(user User) error {
	r.updateUserReceived = append(r.updateUserReceived, user)
	return r.onUpdateUserError
}
