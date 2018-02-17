package mailer

import (
	"io"
	"net/http"
	"reflect"
	"testing"
)

func TestClient_SubscribeUser(t *testing.T) {
	ops := &testClientOperations{}
	config := mailChimpConfig{dc: "x", listID: "y", apiKey: "APIKEY"}

	client := &client{ops: ops, config: config}

	client.SubscribeUser("a@b.com")

	if num := len(ops.received); num != 1 {
		t.Fatalf("Expected to receive 1 request, actually %d", num)
	}

	req := ops.received[0]

	if expected := "POST"; expected != req.Method {
		t.Errorf("Expected method %v, actually %v", expected, req.Method)
	}
	if expected := "https://x.api.mailchimp.com/3.0/lists/y/members"; expected != req.URL.String() {
		t.Errorf("Expected URL %v, actually %v", expected, req.URL.String())
	}
	body, err := read(req.Body)
	if err != nil {
		t.Errorf("Did not expect error %v", err)
	}
	if expected := `{"email address":"a@b.com","status":"subscribed"}`; expected != body {
		t.Errorf("Expected request body %v, actually %v", expected, body)
	}
	if expected := []string{"application/json"}; !reflect.DeepEqual(expected, req.Header["Content-Type"]) {
		t.Errorf("Expected Content-Type %v, actually %v", expected, req.Header["Content-Type"])
	}
	_, password, ok := req.BasicAuth()
	if !ok {
		t.Errorf("Basic auth not ok")
	} else if expected := "APIKEY"; expected != password {
		t.Errorf("Expected basic auth password %v, was %v", expected, password)
	}
}

type testClientOperations struct {
	received []*http.Request
}

func (r *testClientOperations) Do(req *http.Request) (*http.Response, error) {
	r.received = append(r.received, req)
	return nil, nil
}

func read(r io.Reader) (string, error) {
	b := make([]byte, 256)
	n, err := r.Read(b)
	s := string(b[:n])
	return s, err
}
