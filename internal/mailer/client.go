package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Client interface {
	Subscribe(recipient subscription) error
}

type subscription struct {
	email  string
	listID string
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
}

func NewClientConfig(apiKey string) MailChimpConfig {
	return MailChimpConfig{apiKey: apiKey}
}

type mailChimpClient struct {
	log    *Loggers
	ops    clientOperations
	config MailChimpConfig
}

func (c *mailChimpClient) Subscribe(recipient subscription) error {
	// https://developer.mailchimp.com/documentation/mailchimp/guides/manage-subscribers-with-the-mailchimp-api/
	keyParts := strings.Split(c.config.apiKey, "-")
	if len(keyParts) < 2 {
		return fmt.Errorf("API key has no DC suffix")
	}
	dc := keyParts[1]

	url := fmt.Sprintf("https://%s.api.mailchimp.com/3.0/lists/%s/members", dc, recipient.listID)

	payload := postListMembersRequest{Email: recipient.email, Status: "subscribed"}
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

		return fmt.Errorf("error received from %s: HTTP status %d", url, resp.StatusCode)
	}

	return err
}

func NewClient(log *Loggers, config MailChimpConfig) Client {
	return &mailChimpClient{log: log, ops: &http.Client{}, config: config}
}
