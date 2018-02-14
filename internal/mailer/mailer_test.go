package mailer

import (
	"testing"
)

func TestParseSignUp(t *testing.T) {
	str := `{"type":"sign_up","email":"x@y.com"}`

	signUp, err := ParseSignUp(str)

	if err != nil {
		t.Fatalf("error parsing json: %s", err)
	}

	expectedType, expectedEmail := "sign_up", "x@y.com"
	if signUp.Type != expectedType {
		t.Errorf("Expected %s, was %s", expectedType, signUp.Type)
	}
	if signUp.Email != expectedEmail {
		t.Errorf("Expected %s, was %s", expectedEmail, signUp.Email)
	}
}

func TestDoProcess(t *testing.T) {

	message0 := &testMessage{Text: `{"type":"sign_up","email":"x"}`}
	message1 := &testMessage{Text: `{"type":"sign_up","email":"y"}`}

	ms := &testMessageSource{messages: []Message{message0, message1}}
	persister := &testPersister{}

	DoProcess(ms, persister)

	expectedSignUpsCount := 2
	expectedSignUp0Email, expectedSignUp1Email := "x", "y"

	if len(persister.signUps) != expectedSignUpsCount {
		t.Fatalf("Expected %d sign ups, was %d", expectedSignUpsCount, persister.signUps)
	}
	if email := persister.signUps[0].Email; email != expectedSignUp0Email {
		t.Errorf("Expected first persisted sign up email %s, was %s", expectedSignUp0Email, email)
	}
	if message0Processed := ms.processedMessages[message0]; !message0Processed {
		t.Errorf("Expected first Message to be processed")
	}
	if email := persister.signUps[1].Email; email != expectedSignUp1Email {
		t.Errorf("Expected second persisted sign up email %s, was %s", expectedSignUp1Email, email)
	}
	if message1Processed := ms.processedMessages[message1]; !message1Processed {
		t.Errorf("Expected second Message to be processed")
	}
}

// mocks

type testMessageSource struct {
	idx               int
	messages          []Message
	processedMessages map[Message]bool
}

func (ms *testMessageSource) GetNextMessage() (Message, error) {
	var msg Message
	if ms.idx < len(ms.messages) {
		msg = ms.messages[ms.idx]
		ms.idx++
	}
	return msg, nil
}

func (ms *testMessageSource) MessageProcessed(msg Message) error {
	if ms.processedMessages == nil {
		ms.processedMessages = map[Message]bool{}
	}
	ms.processedMessages[msg] = true
	return nil
}

type testMessage struct {
	Text string
}

func (msg *testMessage) GetText() string {
	return msg.Text
}

type testPersister struct {
	signUps []SignUp
}

func (persister *testPersister) InsertSignUp(signUp SignUp) error {
	persister.signUps = append(persister.signUps, signUp)
	return nil
}
