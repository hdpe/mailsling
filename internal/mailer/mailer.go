package mailer

import (
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
)

type SignUp struct {
	Type  string `json:"type"`
	Email string `json:"email"`
}

func DoProcess(ms MessageSource, persister Persister) error {
	for {
		msg, err := ms.GetNextMessage()
		if err != nil {
			return err
		} else if msg == nil {
			break
		}

		signUp, err := ParseSignUp(msg.GetText())
		if err != nil {
			return err
		}

		err = persister.InsertSignUp(signUp)
		if err != nil {
			return err
		}

		err = ms.MessageProcessed(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

func ParseSignUp(str string) (SignUp, error) {
	var msg SignUp

	err := json.Unmarshal([]byte(str), &msg)

	return msg, err
}

