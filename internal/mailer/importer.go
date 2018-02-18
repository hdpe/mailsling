package mailer

import (
	"encoding/json"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type SignUpMessage struct {
	Type  string `json:"type"`
	Email string `json:"email"`
}

type Importer struct {
	ms        MessageSource
	persister Persister
}

func (r *Importer) DoProcess() error {
	for {
		msg, err := r.ms.GetNextMessage()
		if err != nil {
			return fmt.Errorf("couldn't get next message from queue: %v", err)
		} else if msg == nil {
			break
		}

		signUp, err := parseSignUp(msg.GetText())
		if err != nil {
			return fmt.Errorf("couldn't parse sign up from message %q: %v", msg.GetText(), err)
		}

		err = r.persister.InsertSignUp(SignUp{Email: signUp.Email})
		if err != nil {
			return fmt.Errorf("couldn't insert sign up to DB: %v", err)
		}

		err = r.ms.MessageProcessed(msg)
		if err != nil {
			return fmt.Errorf("couldn't mark message processed: %v", err)
		}
	}

	return nil
}

func parseSignUp(str string) (SignUpMessage, error) {
	var msg SignUpMessage

	err := json.Unmarshal([]byte(str), &msg)

	return msg, err
}
