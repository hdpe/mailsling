package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
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

type clientOperations interface {
	Do(req *http.Request) (*http.Response, error)
}

type MailChimpConfig struct {
	apiKey string
	listID string
}

func NewClientConfig(apiKey string, listID string) MailChimpConfig {
	return MailChimpConfig{apiKey: apiKey, listID: listID}
}

type mailChimpClient struct {
	log    *Loggers
	ops    clientOperations
	config MailChimpConfig
}

func (c *mailChimpClient) SubscribeUser(user User) error {
	// https://developer.mailchimp.com/documentation/mailchimp/guides/manage-subscribers-with-the-mailchimp-api/
	keyParts := strings.Split(c.config.apiKey, "-")
	if len(keyParts) < 2 {
		return fmt.Errorf("API key has no DC suffix")
	}
	dc := keyParts[1]

	url := fmt.Sprintf("https://%s.api.mailchimp.com/3.0/lists/%s/members", dc, c.config.listID)

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
	req.SetBasicAuth("IGNORED", c.config.apiKey)

	resp, err := c.ops.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b := &bytes.Buffer{}
		b.ReadFrom(resp.Body)
		c.log.Error.Println(string(b.Bytes()))

		return fmt.Errorf("error received from server: HTTP status %d", resp.StatusCode)
	}

	return err
}

func NewClient(log *Loggers, config MailChimpConfig) Client {
	return &mailChimpClient{log: log, ops: &http.Client{}, config: config}
}
