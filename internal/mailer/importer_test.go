package mailer

import "testing"

func TestParseSignUp(t *testing.T) {
	str := `{"type":"sign_up","email":"x@y.com"}`

	signUp, err := parseSignUp(str)

	if err != nil {
		t.Fatalf("error parsing json: %s", err)
	}

	if expected, actual := "sign_up", signUp.Type; expected != actual {
		t.Errorf("Expected %s, was %s", expected, actual)
	}
	if expected, actual := "x@y.com", signUp.Email; expected != actual {
		t.Errorf("Expected %s, was %s", expected, actual)
	}
}

func TestDoProcess(t *testing.T) {

	message0 := &testMessage{Text: `{"type":"sign_up","email":"x"}`}
	message1 := &testMessage{Text: `{"type":"sign_up","email":"y"}`}

	ms := &testMessageSource{messages: []Message{message0, message1}}
	persister := &testPersister{}
	importer := &Importer{ms: ms, persister: persister}

	importer.DoProcess()

	if expected, actual := 2, len(persister.signUps); expected != actual {
		t.Fatalf("Expected %d sign ups, was %d", expected, actual)
	}
	if expected, actual := "x", persister.signUps[0].Email; expected != actual {
		t.Errorf("Expected first persisted sign up email %s, was %s", expected, actual)
	}
	if processed := ms.processedMessages[message0]; !processed {
		t.Errorf("Expected first Message to be processed")
	}
	if expected, actual := "y", persister.signUps[1].Email; expected != actual {
		t.Errorf("Expected second persisted sign up email %s, was %s", expected, actual)
	}
	if processed := ms.processedMessages[message1]; !processed {
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
