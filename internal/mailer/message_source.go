package mailer

import (
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

type MessageSource interface {
	GetNextMessage() (Message, error)
	MessageProcessed(message Message) error
}

type Message interface {
	GetText() string
}

type SQSMessageSource struct {
	sqsClient sqsiface.SQSAPI
	url       string
	messages  []Message
}

func NewSQSMessageSource(queueUrl string) (*SQSMessageSource, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	ms := &SQSMessageSource{sqsClient: sqs.New(sess), url: queueUrl}
	return ms, nil
}

func (ms *SQSMessageSource) GetNextMessage() (Message, error) {
	if next := ms.dequeue(); next != nil {
		return next, nil
	}
	out, err := ms.sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{QueueUrl: &ms.url})
	if out == nil || err != nil {
		return nil, err
	}
	log.Printf("Got %d messages", len(out.Messages))
	for _, msg := range out.Messages {
		ms.messages = append(ms.messages, &sqsMessage{delegate: msg})
	}
	return ms.dequeue(), nil
}

func (ms *SQSMessageSource) MessageProcessed(message Message) error {
	handle := message.(*sqsMessage).delegate.ReceiptHandle
	_, err := ms.sqsClient.DeleteMessage(&sqs.DeleteMessageInput{QueueUrl: &ms.url, ReceiptHandle: handle})
	return err
}

func (ms *SQSMessageSource) dequeue() Message {
	if len(ms.messages) > 0 {
		result := ms.messages[0]
		ms.messages = ms.messages[1:]
		return result
	}
	return nil
}

type sqsMessage struct {
	delegate *sqs.Message
}

func (ms *sqsMessage) GetText() string {
	return *ms.delegate.Body
}
