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

func TestMailChimpClient_Subscribe(t *testing.T) {
	ops := &testClientOperations{}
	config := MailChimpConfig{listID: "y", apiKey: "APIKEY-dc"}

	client := &mailChimpClient{ops: ops, config: config}

	client.Subscribe(Recipient{Email: "a@b.com"})

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

func TestMailChimpClient_SubscribeErrors(t *testing.T) {
	testCases := []struct {
		label    string
		apiKey   string
		onDo     func() (*http.Response, error)
		expected string
	}{
		{
			label:  "on API key has no dc suffix",
			apiKey: "x",
			onDo: func() (*http.Response, error) {
				return newClientTestResponse(200), nil
			},
			expected: "API key has no DC suffix",
		},
		{
			label:  "on http.NewRequest",
			apiKey: "-%z",
			onDo: func() (*http.Response, error) {
				return nil, nil
			},
			expected: "error creating request",
		},
		{
			label:  "on clientOperations.Do",
			apiKey: "-x",
			onDo: func() (*http.Response, error) {
				return nil, errors.New("")
			},
			expected: "error sending request",
		},
		{
			label:  "on non-OK response",
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

		client := &mailChimpClient{log: NOOPLog, ops: ops, config: config}

		err := client.Subscribe(Recipient{Email: "a@b.com"})

		if err == nil || strings.Index(fmt.Sprintf("%s", err), tc.expected) != 0 {
			t.Errorf("Expected error %s %q, actually %q", tc.label, tc.expected, err)
		}
	}
}

type testClientOperations struct {
	received []*http.Request
	onDo     func() (*http.Response, error)
}

func (ops *testClientOperations) Do(req *http.Request) (*http.Response, error) {
	ops.received = append(ops.received, req)
	if ops.onDo != nil {
		return ops.onDo()
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
