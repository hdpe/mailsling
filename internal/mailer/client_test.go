package mailer

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestMailChimpOperations_Execute(t *testing.T) {
	clientOps := &testClientOperations{}
	config := MailChimpConfig{apiKey: "APIKEY-dc"}

	ops := &mailChimpOperations{ops: clientOps, config: config}

	ops.execute("POST", "/path", postListMemberRequest{Email: "a@b.com", Status: "c"})

	if num := len(clientOps.received); num != 1 {
		t.Fatalf("invoked Do %d times, want 1", num)
	}

	req := clientOps.received[0]

	if actual, expected := req.Method, "POST"; actual != expected {
		t.Errorf("method got %q, want %q", actual, expected)
	}
	if actual, expected := req.URL.String(), "https://dc.api.mailchimp.com/3.0/path"; actual != expected {
		t.Errorf("URL got %v, want %v", actual, expected)
	}
	body, err := read(req.Body)
	if err != nil {
		t.Errorf("read error got %q, want nil", err)
	}
	if expected := `{"email_address":"a@b.com","status":"c"}`; body != expected {
		t.Errorf("request body got %v, want %v", body, expected)
	}
	if actual, expected := req.Header["Content-Type"], []string{"application/json"}; !reflect.DeepEqual(actual, expected) {
		t.Errorf("Content-Type header got %q, actually %q", actual, expected)
	}
	_, password, ok := req.BasicAuth()
	if !ok {
		t.Errorf("basic auth not ok")
	} else if expected := "APIKEY-dc"; password != expected {
		t.Errorf("basic auth password got %q, want %q", password, expected)
	}
}

func TestMailChimpOperations_ExecuteErrors(t *testing.T) {
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
			expected: "error received from",
		},
	}

	for _, tc := range testCases {
		clientOps := &testClientOperations{onDo: tc.onDo}
		config := MailChimpConfig{apiKey: tc.apiKey}

		ops := &mailChimpOperations{log: NOOPLog, ops: clientOps, config: config}

		err := ops.execute("POST", "/path", subscription{email: "a@b.com"})

		if !errorMessageStartsWith(err, tc.expected) {
			t.Errorf("%v: result error got %q, want %q", tc.label, err, tc.expected)
		}
	}
}

func TestMailChimpClient_SubscribeAndUnsubscribe(t *testing.T) {
	testCases := []struct {
		label string

		testMethod   func(c *mailChimpClient, s subscription) error
		subscription subscription

		executeInvoked bool
		onExecute      func(method string, url string, entity interface{}) error

		expected error
	}{
		{
			label: "subscribe invokes execute",

			testMethod: func(c *mailChimpClient, s subscription) error {
				return c.Subscribe(s)
			},
			subscription: subscription{email: "a@b.com", listID: "c"},

			executeInvoked: true,
			onExecute: func(method string, url string, entity interface{}) error {
				if expected := "POST"; method != expected {
					t.Errorf("subscribe invokes execute: ops Execute got %q, want %q", method, expected)
				}
				if expected := "/lists/c/members"; url != expected {
					t.Errorf("subscribe invokes execute: ops Execute got %q, want %q", url, expected)
				}
				expectedEntity := postListMemberRequest{Email: "a@b.com", Status: "subscribed"}
				if entity != expectedEntity {
					t.Errorf("subscribe invokes execute: ops Execute got %q, want %q", entity, expectedEntity)
				}
				return nil
			},

			expected: nil,
		},
		{
			label: "returns error on subscribe error",

			testMethod: func(c *mailChimpClient, s subscription) error {
				return c.Subscribe(s)
			},
			subscription: subscription{email: "a@b.com", listID: "c"},

			executeInvoked: true,
			onExecute: func(method string, url string, entity interface{}) error {
				return errors.New("x")
			},

			expected: errors.New("x"),
		},
		{
			label: "unsubscribe invokes execute",

			testMethod: func(c *mailChimpClient, s subscription) error {
				return c.Unsubscribe(s)
			},
			subscription: subscription{email: "a@b.com", listID: "c"},

			executeInvoked: true,
			onExecute: func(method string, url string, entity interface{}) error {
				if expected := "PATCH"; method != expected {
					t.Errorf("unsubscribe invokes execute: ops Execute got %q, want %q", method, expected)
				}
				// 357a20e8c56e69d6f9734d23ef9517e8 = md5 of a@b.com
				if expected := "/lists/c/members/357a20e8c56e69d6f9734d23ef9517e8"; url != expected {
					t.Errorf("unsubscribe invokes execute: ops Execute got %q, want %q", url, expected)
				}
				expectedEntity := patchListMemberStatusRequest{Status: "unsubscribed"}
				if entity != expectedEntity {
					t.Errorf("unsubscribe invokes execute: ops Execute got %q, want %q", entity, expectedEntity)
				}
				return nil
			},

			expected: nil,
		},
		{
			label: "returns error on unsubscribe error",

			testMethod: func(c *mailChimpClient, s subscription) error {
				return c.Unsubscribe(s)
			},
			subscription: subscription{email: "a@b.com", listID: "c"},

			executeInvoked: true,
			onExecute: func(method string, url string, entity interface{}) error {
				return errors.New("x")
			},

			expected: errors.New("x"),
		},
	}

	for _, tc := range testCases {
		ops := &testMailChimpOperations{onExecute: tc.onExecute}

		client := &mailChimpClient{ops: ops}

		err := tc.testMethod(client, tc.subscription)

		if ops.executeInvoked != tc.executeInvoked {
			t.Errorf("%v: execute invoked got %v, want %v", tc.label, ops.executeInvoked, tc.executeInvoked)
		}
		if !errorEquals(err, tc.expected) {
			t.Errorf("%v: result got %q, want %q", tc.label, err, tc.expected)
		}
	}
}

func TestClientNotifier_Notify(t *testing.T) {
	testSubscription := subscription{email: "x", listID: "y"}

	testCases := []struct {
		label string

		status RecipientStatus

		subscribeInvoked bool
		onSubscribe      func(s subscription) error

		unsubscribeInvoked bool
		onUnsubscribe      func(s subscription) error

		expectedStatus RecipientStatus
		expectedError  error
	}{
		{
			label: "on status = new",

			status: RecipientStatuses.Get("new"),

			subscribeInvoked: true,
			onSubscribe: func(s subscription) error {
				if s != testSubscription {
					t.Errorf("on status = new: client Subscribe got %q, want %q", s, testSubscription)
				}
				return nil
			},

			expectedStatus: RecipientStatuses.Get("subscribed"),
			expectedError:  nil,
		},
		{
			label: "returns error on subscribe error",

			status: RecipientStatuses.Get("new"),

			subscribeInvoked: true,
			onSubscribe: func(s subscription) error {
				return errors.New("x")
			},

			expectedStatus: RecipientStatuses.None,
			expectedError:  errors.New("x"),
		},
		{
			label: "on status = unsubscribing",

			status: RecipientStatuses.Get("unsubscribing"),

			unsubscribeInvoked: true,
			onUnsubscribe: func(s subscription) error {
				if s != testSubscription {
					t.Errorf("on status = unsubscribing: client Unsubscribe got %q, want %q", s, testSubscription)
				}
				return nil
			},

			expectedStatus: RecipientStatuses.Get("unsubscribed"),
			expectedError:  nil,
		},
		{
			label: "returns error on unsubscribe error",

			status: RecipientStatuses.Get("unsubscribing"),

			unsubscribeInvoked: true,
			onUnsubscribe: func(s subscription) error {
				return errors.New("x")
			},

			expectedStatus: RecipientStatuses.None,
			expectedError:  errors.New("x"),
		},
	}

	for _, tc := range testCases {
		client := newNotifierTestClient(tc.onSubscribe, tc.onUnsubscribe)
		n := &clientNotifier{client: client}

		result, err := n.Notify(testSubscription, tc.status)

		if client.subscribeInvoked != tc.subscribeInvoked {
			t.Errorf("%v: subscribe invoked got %v, want %v", tc.label, client.subscribeInvoked, tc.subscribeInvoked)
		}
		if result != tc.expectedStatus {
			t.Errorf("%v: result status got %v, want %v", tc.label, result, tc.expectedStatus)
		}
		if !reflect.DeepEqual(err, tc.expectedError) {
			t.Errorf("%v: result error got %v, want %v", tc.label, err, tc.expectedError)
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

type notifierTestClient struct {
	subscribeInvoked bool
	onSubscribe      func(s subscription) error

	unsubscribeInvoked bool
	onUnsubscribe      func(s subscription) error
}

func (c *notifierTestClient) Subscribe(s subscription) error {
	c.subscribeInvoked = true
	return c.onSubscribe(s)
}

func (c *notifierTestClient) Unsubscribe(s subscription) error {
	c.unsubscribeInvoked = true
	return c.onUnsubscribe(s)
}

func newNotifierTestClient(onSubscribe func(s subscription) error, onUnsubscribe func(s subscription) error) *notifierTestClient {
	c := &notifierTestClient{
		onSubscribe: func(s subscription) error {
			return nil
		},
		onUnsubscribe: func(s subscription) error {
			return nil
		},
	}
	if onSubscribe != nil {
		c.onSubscribe = onSubscribe
	}
	if onUnsubscribe != nil {
		c.onUnsubscribe = onUnsubscribe
	}
	return c
}

type testMailChimpOperations struct {
	executeInvoked bool
	onExecute      func(method string, url string, entity interface{}) error
}

func (o *testMailChimpOperations) execute(method string, url string, entity interface{}) error {
	o.executeInvoked = true
	return o.onExecute(method, url, entity)
}
