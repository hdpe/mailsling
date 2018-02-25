package mailer

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

func TestSqsMessageSource_GetNextMessageInvokesAWSApi(t *testing.T) {
	client := &testSqsClient{}
	ms := SQSMessageSource{log: NOOPLog, url: "http://x", sqsClient: client}

	_, _ = ms.GetNextMessage()

	expected := sqs.ReceiveMessageInput{QueueUrl: strptr("http://x")}

	if received := client.receiveMessageReceived; !reflect.DeepEqual(*received, expected) {
		t.Fatalf("Expected to have requested receive %v, actually requested receive %v", expected, received)
	}
}

func TestSqsMessageSource_GetNextMessageReturnsMessages(t *testing.T) {
	client := &testSqsClient{receiveMessageResultMessages: [][]sqs.Message{
		{{Body: strptr("x")}},
	}}
	ms := SQSMessageSource{log: NOOPLog, sqsClient: client}

	next, err := ms.GetNextMessage()

	if err != nil {
		t.Fatalf("Error was returned")
	}
	if next == nil {
		t.Fatalf("Message was nil")
	}
	if expected, txt := "x", next.GetText(); txt != expected {
		t.Errorf("Expected Message body %s, was %s", expected, txt)
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
	ms := SQSMessageSource{log: NOOPLog, url: "http://x", sqsClient: client}

	_ = ms.MessageProcessed(&sqsMessage{delegate: &sqs.Message{ReceiptHandle: strptr("y")}})

	expected := sqs.DeleteMessageInput{QueueUrl: strptr("http://x"), ReceiptHandle: strptr("y")}

	if received := client.deleteMessageReceived; !reflect.DeepEqual(*received, expected) {
		t.Fatalf("Expected to have requested delete %v, actually requested delete %v", expected, received)
	}
}

// mocks & utils

type testSqsClient struct {
	sqsiface.SQSAPI
	receiveMessageRequestIndex   int
	receiveMessageResultMessages [][]sqs.Message
	receiveMessageReceived       *sqs.ReceiveMessageInput
	deleteMessageReceived        *sqs.DeleteMessageInput
}

func (c *testSqsClient) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	c.receiveMessageReceived = input
	if c.receiveMessageRequestIndex >= len(c.receiveMessageResultMessages) {
		return &sqs.ReceiveMessageOutput{}, nil
	}
	var result []*sqs.Message
	for i := 0; i < len(c.receiveMessageResultMessages[c.receiveMessageRequestIndex]); i++ {
		result = append(result, &c.receiveMessageResultMessages[c.receiveMessageRequestIndex][i])
	}
	c.receiveMessageRequestIndex++
	return &sqs.ReceiveMessageOutput{Messages: result}, nil
}

func (c *testSqsClient) DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	c.deleteMessageReceived = input
	return nil, nil
}

func strptr(in string) *string {
	return &in
}
