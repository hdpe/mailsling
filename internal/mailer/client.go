package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Client interface {
	SubscribeUser(user User) error
}

type postListMembersRequest struct {
	Email  string `json:"email_address"`
	Status string `json:"status"`
}

func NewClientConfig(apiKey string, listID string) *MailChimpConfig {
	return &MailChimpConfig{apiKey: apiKey, listID: listID}
}

type clientOperations interface {
	Do(req *http.Request) (*http.Response, error)
}

type mailChimpClient struct {
	ops    clientOperations
	config MailChimpConfig
}

type MailChimpConfig struct {
	apiKey string
	listID string
}

func (r MailChimpConfig) NewClient() Client {
	return &mailChimpClient{ops: &http.Client{}, config: r}
}

func (r *mailChimpClient) SubscribeUser(user User) error {
	// https://developer.mailchimp.com/documentation/mailchimp/guides/manage-subscribers-with-the-mailchimp-api/
	keyParts := strings.Split(r.config.apiKey, "-")
	if len(keyParts) < 2 {
		return fmt.Errorf("API key has no DC suffix")
	}
	dc := keyParts[1]

	url := fmt.Sprintf("https://%s.api.mailchimp.com/3.0/lists/%s/members", dc, r.config.listID)

	payload := postListMembersRequest{Email: user.Email, Status: "subscribed"}
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
		b := &bytes.Buffer{}
		b.ReadFrom(resp.Body)
		log.Println(string(b.Bytes()))

		return fmt.Errorf("error received from server: HTTP status %d", resp.StatusCode)
	}

	return err
}
