package mailer

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

func TestSqsMessageSource_GetNextMessageInvokesAWSApi(t *testing.T) {
	client := &testSqsClient{}
	ms := SQSMessageSource{url: "http://x", sqsClient: client}

	_, _ = ms.GetNextMessage()

	expected := sqs.ReceiveMessageInput{QueueUrl: strptr("http://x")}

	if received := client.receivedReceiveMessageInput; !reflect.DeepEqual(*received, expected) {
		t.Fatalf("Expected to have requested receive %v, actually requested receive %v", expected, received)
	}
}

func TestSqsMessageSource_GetNextMessageReturnsMessages(t *testing.T) {
	msgBody0 := "x"
	client := &testSqsClient{messagesPerRequest: [][]sqs.Message{
		{{Body: strptr("x")}},
	}}
	ms := SQSMessageSource{sqsClient: client}

	next, err := ms.GetNextMessage()

	if err != nil {
		t.Fatalf("Error was returned")
	}
	if next == nil {
		t.Fatalf("Message was nil")
	}
	if txt := next.GetText(); txt != msgBody0 {
		t.Errorf("Expected Message body %s, was %s", msgBody0, txt)
	}

	next, err = ms.GetNextMessage()

	if err != nil {
		t.Fatalf("Error was returned")
	}
	if next != nil {
		t.Fatalf("Expected Message to be nil")
	}
}

func TestSqsMessageSource_MessageProcessed(t *testing.T) {
	client := &testSqsClient{}
	ms := SQSMessageSource{url: "http://x", sqsClient: client}

	_ = ms.MessageProcessed(&sqsMessage{delegate: &sqs.Message{ReceiptHandle: strptr("y")}})

	expected := sqs.DeleteMessageInput{QueueUrl: strptr("http://x"), ReceiptHandle: strptr("y")}

	if received := client.receivedDeleteMessageInput; !reflect.DeepEqual(*received, expected) {
		t.Fatalf("Expected to have requested delete %v, actually requested delete %v", expected, received)
	}
}

// mocks & utils

type testSqsClient struct {
	sqsiface.SQSAPI
	messagesPerRequest          [][]sqs.Message
	requestIndex                int
	receivedReceiveMessageInput *sqs.ReceiveMessageInput
	receivedDeleteMessageInput  *sqs.DeleteMessageInput
}

func (c *testSqsClient) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	c.receivedReceiveMessageInput = input
	if c.requestIndex >= len(c.messagesPerRequest) {
		return nil, nil
	}
	var result []*sqs.Message
	for i := 0; i < len(c.messagesPerRequest[c.requestIndex]); i++ {
		result = append(result, &c.messagesPerRequest[c.requestIndex][i])
	}
	c.requestIndex++
	return &sqs.ReceiveMessageOutput{Messages: result}, nil
}

func (c *testSqsClient) DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	c.receivedDeleteMessageInput = input
	return nil, nil
}

func strptr(in string) *string {
	return &in
}
