package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type clientOperations interface {
	Do(req *http.Request) (*http.Response, error)
}

type client struct {
	ops    clientOperations
	config mailChimpConfig
}

type mailChimpConfig struct {
	dc     string
	apiKey string
	listID string
}

type postListMembersRequest struct {
	Email  string `json:"email address"`
	Status string `json:"status"`
}

func newClient(config mailChimpConfig) *client {
	return &client{ops: &http.Client{}, config: config}
}

func (r *client) SubscribeUser(email string) error {
	// https://developer.mailchimp.com/documentation/mailchimp/guides/manage-subscribers-with-the-mailchimp-api/
	req := postListMembersRequest{Email: email, Status: "subscribed"}
	url := fmt.Sprintf("https://%s.api.maidlchimp.com/3.0/lists/%s/members", r.config.dc, r.config.listID)
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	req2, err := http.NewRequest("POST", url, bytes.NewReader(b))
	req2.Header["Content-Type"] = []string{"application/json"}
	req2.SetBasicAuth("IGNORED", r.config.apiKey)
	if err != nil {
		return err
	}
	_, err = r.ops.Do(req2)
	return err
}
