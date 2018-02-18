package mailer

import (
	"database/sql"
	"fmt"
)

type Persister interface {
	InsertSignUp(signUp SignUp) error
}

type DBPersister struct {
	Db *sql.DB
}

type SignUp struct {
	Email string
}

func (persister *DBPersister) InsertSignUp(signUp SignUp) error {
	tx, err := persister.Db.Begin()
	if err != nil {
		return fmt.Errorf("couldn't start tx: %v", err)
	}
	_, err = tx.Exec("insert into sign_ups (email) values (?)", signUp.Email)
	if err != nil {
		return fmt.Errorf("couldn't perform insert: %v", err)
	}
	err = tx.Commit()
	return fmt.Errorf("couldn't commit tx: %v", err)
}

func NewPersister(dsn string) (*DBPersister, error) {
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		return nil, fmt.Errorf("couldn't open connection to %q: %v", dsn, err)
	}

	return &DBPersister{Db: db}, err
}
