package mailer

import "database/sql"

type Persister interface {
	InsertSignUp(signUp SignUp) error
}

type DBPersister struct {
	Db *sql.DB
}

func NewPersister(dsn string) (*DBPersister, error) {
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		return nil, err
	}

	return &DBPersister{Db: db}, err
}

func (persister *DBPersister) InsertSignUp(signUp SignUp) error {
	tx, err := persister.Db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("insert into sign_ups (email) values (?)", signUp.Email)
	if err != nil {
		return err
	}
	return tx.Commit()
}
