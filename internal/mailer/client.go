package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client interface {
	SubscribeUser(signUp SignUpMessage) error
}

type postListMembersRequest struct {
	Email  string `json:"email address"`
	Status string `json:"status"`
}

func NewClient(config MailChimpConfig) Client {
	return &mailChimpClient{ops: &http.Client{}, config: config}
}

type clientOperations interface {
	Do(req *http.Request) (*http.Response, error)
}

type mailChimpClient struct {
	ops    clientOperations
	config MailChimpConfig
}

type MailChimpConfig struct {
	dc     string
	apiKey string
	listID string
}

func (r *mailChimpClient) SubscribeUser(signUp SignUpMessage) error {
	// https://developer.mailchimp.com/documentation/mailchimp/guides/manage-subscribers-with-the-mailchimp-api/
	url := fmt.Sprintf("https://%s.api.mailchimp.com/3.0/lists/%s/members", r.config.dc, r.config.listID)

	payload := postListMembersRequest{Email: signUp.Email, Status: "subscribed"}
	b, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header["Content-Type"] = []string{"application/json"}
	req.SetBasicAuth("IGNORED", r.config.apiKey)

	resp, err := r.ops.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("error received from server: HTTP status %d", resp.StatusCode)
	}

	return err
}
