package mailer

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client interface {
	Subscribe(s subscription) error
	Unsubscribe(s subscription) error
}

type subscription struct {
	email  string
	listID string
}

type postListMemberRequest struct {
	Email  string `json:"email_address"`
	Status string `json:"status"`
}

type patchListMemberStatusRequest struct {
	Status string `json:"status"`
}

type clientOperations interface {
	Do(req *http.Request) (*http.Response, error)
}

type mailChimpExecutor interface {
	execute(method string, url string, entity interface{}) error
}

type mailChimpOperations struct {
	log    *Loggers
	ops    clientOperations
	config MailChimpConfig
}

func (o *mailChimpOperations) execute(method string, url string, entity interface{}) error {
	// https://developer.mailchimp.com/documentation/mailchimp/guides/manage-subscribers-with-the-mailchimp-api/
	keyParts := strings.Split(o.config.apiKey, "-")
	if len(keyParts) < 2 {
		return fmt.Errorf("API key has no DC suffix")
	}
	dc := keyParts[1]

	url = fmt.Sprintf("https://%s.api.mailchimp.com/3.0%s", dc, url)

	b, err := json.Marshal(entity)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header["Content-Type"] = []string{"application/json"}
	req.SetBasicAuth("IGNORED", o.config.apiKey)

	resp, err := o.ops.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b := &bytes.Buffer{}
		b.ReadFrom(resp.Body)
		o.log.Error.Println(string(b.Bytes()))

		return fmt.Errorf("error received from %s: HTTP status %d", url, resp.StatusCode)
	}

	return err
}

type MailChimpConfig struct {
	apiKey string
}

func NewClientConfig(apiKey string) MailChimpConfig {
	return MailChimpConfig{apiKey: apiKey}
}

type mailChimpClient struct {
	ops mailChimpExecutor
}

func (c *mailChimpClient) Subscribe(s subscription) error {
	url := fmt.Sprintf("/lists/%s/members", s.listID)
	request := postListMemberRequest{Email: s.email, Status: "subscribed"}

	return c.ops.execute("POST", url, request)
}

func (c *mailChimpClient) Unsubscribe(s subscription) error {
	id := getSubscriberID(s)

	url := fmt.Sprintf("/lists/%s/members/%s", s.listID, id)
	request := patchListMemberStatusRequest{Status: "unsubscribed"}

	return c.ops.execute("PATCH", url, request)
}

func getSubscriberID(s subscription) string {
	h := md5.New()
	io.WriteString(h, s.email)
	id := hex.EncodeToString(h.Sum(nil))
	return id
}

func NewClient(log *Loggers, config MailChimpConfig) Client {
	return &mailChimpClient{ops: &mailChimpOperations{log: log, ops: &http.Client{}, config: config}}
}

type clientNotifier struct {
	client Client
}

func (n *clientNotifier) Notify(s subscription, currentStatus RecipientStatus) (result RecipientStatus, err error) {

	if currentStatus == RecipientStatuses.Get("new") {
		err = n.client.Subscribe(s)
		if err == nil {
			result = RecipientStatuses.Get("subscribed")
		}
	} else if currentStatus == RecipientStatuses.Get("unsubscribing") {
		err = n.client.Unsubscribe(s)
		if err == nil {
			result = RecipientStatuses.Get("unsubscribed")
		}
	}

	if err != nil {
		result = RecipientStatuses.None
	}

	return
}
