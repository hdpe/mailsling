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
		t.Fatalf("invoked ReceiveMessage got %v, want %v", *received, expected)
	}
}

func TestSqsMessageSource_GetNextMessageReturnsMessages(t *testing.T) {
	client := &testSqsClient{receiveMessageResultMessages: [][]sqs.Message{
		{{Body: strptr("x")}},
	}}
	ms := SQSMessageSource{log: NOOPLog, sqsClient: client}

	next, err := ms.GetNextMessage()

	if err != nil {
		t.Fatalf("error got %q, want nil", err)
	}
	if next == nil {
		t.Fatalf("message was nil")
	}
	if txt, expected := next.GetText(), "x"; txt != expected {
		t.Errorf("messag body got %q, want %q", txt, expected)
	}

	next, err = ms.GetNextMessage()

	if err != nil {
		t.Fatalf("error got %q, want nil (2nd call)", err)
	}
	if next != nil {
		t.Fatalf("message was nil")
	}
}

func TestSqsMessageSource_MessageProcessed(t *testing.T) {
	client := &testSqsClient{}
	ms := SQSMessageSource{log: NOOPLog, url: "http://x", sqsClient: client}

	res := ms.MessageProcessed(&sqsMessage{delegate: &sqs.Message{ReceiptHandle: strptr("y")}})

	expected := sqs.DeleteMessageInput{QueueUrl: strptr("http://x"), ReceiptHandle: strptr("y")}

	if res != nil {
		t.Errorf("error got %q, want nil", res)
	}
	if received := client.deleteMessageReceived; !reflect.DeepEqual(*received, expected) {
		t.Errorf("invoked DeleteMessage got %v, want %v", *received, expected)
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
