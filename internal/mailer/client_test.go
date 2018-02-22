package mailer

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestMailChimpClient_SubscribeUser(t *testing.T) {
	ops := &testClientOperations{}
	config := MailChimpConfig{listID: "y", apiKey: "APIKEY-dc"}

	client := &mailChimpClient{ops: ops, config: config}

	client.SubscribeUser(User{Email: "a@b.com"})

	if num := len(ops.received); num != 1 {
		t.Fatalf("Expected to receive 1 request, actually %d", num)
	}

	req := ops.received[0]

	if expected, actual := "POST", req.Method; expected != actual {
		t.Errorf("Expected method %v, actually %v", expected, actual)
	}
	if expected, actual := "https://dc.api.mailchimp.com/3.0/lists/y/members", req.URL.String(); expected != actual {
		t.Errorf("Expected URL %v, actually %v", expected, actual)
	}
	body, err := read(req.Body)
	if err != nil {
		t.Errorf("Did not expect error %v", err)
	}
	if expected := `{"email_address":"a@b.com","status":"subscribed"}`; expected != body {
		t.Errorf("Expected request body %v, actually %v", expected, body)
	}
	if expected, actual := []string{"application/json"}, req.Header["Content-Type"]; !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected Content-Type %v, actually %v", expected, actual)
	}
	_, password, ok := req.BasicAuth()
	if !ok {
		t.Errorf("Basic auth not ok")
	} else if expected := "APIKEY-dc"; expected != password {
		t.Errorf("Expected basic auth password %v, was %v", expected, password)
	}
}

func TestMailChimpClient_SubscribeUserErrors(t *testing.T) {
	testCases := []struct {
		d        string
		apiKey   string
		onDo     func() (*http.Response, error)
		expected string
	}{
		{
			d:      "on API key has no dc suffix",
			apiKey: "x",
			onDo: func() (*http.Response, error) {
				return newClientTestResponse(200), nil
			},
			expected: "API key has no DC suffix",
		},
		{
			d:      "on http.NewRequest",
			apiKey: "-%z",
			onDo: func() (*http.Response, error) {
				return nil, nil
			},
			expected: "error creating request",
		},
		{
			d:      "on clientOperations.Do",
			apiKey: "-x",
			onDo: func() (*http.Response, error) {
				return nil, errors.New("")
			},
			expected: "error sending request",
		},
		{
			d:      "on non-OK response",
			apiKey: "-x",
			onDo: func() (*http.Response, error) {
				return newClientTestResponse(400), nil
			},
			expected: "error received from server",
		},
	}

	for _, tc := range testCases {
		ops := &testClientOperations{onDo: tc.onDo}
		config := MailChimpConfig{apiKey: tc.apiKey}

		client := &mailChimpClient{ops: ops, config: config}

		err := client.SubscribeUser(User{Email: "a@b.com"})

		if err == nil || strings.Index(fmt.Sprintf("%s", err), tc.expected) != 0 {
			t.Errorf("Expected error %s %q, actually %q", tc.d, tc.expected, err)
		}
	}
}

type testClientOperations struct {
	received []*http.Request
	onDo     func() (*http.Response, error)
}

func (r *testClientOperations) Do(req *http.Request) (*http.Response, error) {
	r.received = append(r.received, req)
	if r.onDo != nil {
		return r.onDo()
	}
	return newClientTestResponse(200), nil
}

func read(r io.Reader) (string, error) {
	b := make([]byte, 256)
	n, err := r.Read(b)
	s := string(b[:n])
	return s, err
}

func newClientTestResponse(statusCode int) *http.Response {
	return &http.Response{StatusCode: statusCode, Body: &clientTestResponseBody{strings.NewReader("")}}
}

type clientTestResponseBody struct {
	io.Reader
}

func (r *clientTestResponseBody) Close() error {
	return nil
}
