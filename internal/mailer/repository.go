package mailer

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type Repository interface {
	GetNewRecipients() ([]Recipient, error)
	GetRecipientByEmail(string) (recipient Recipient, found bool, err error)
	InsertRecipient(Recipient) error
	UpdateRecipient(Recipient) error
	Close() error
}

type DBRepository struct {
	Db *sql.DB
}

func (r *DBRepository) GetNewRecipients() (result []Recipient, err error) {
	rows, err := r.Db.Query("select id, email, status from recipients where status = ?",
		RecipientStatuses.Get("new"))

	if err != nil {
		err = fmt.Errorf("couldn't get row: %v", err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var rec Recipient
		rec, err = mapRow(rows)
		if err != nil {
			err = fmt.Errorf("error retrieving row: %v", err)
			return
		}
		result = append(result, rec)
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("error iterating rows: %v", err)
	}

	return result, err
}

func (r *DBRepository) GetRecipientByEmail(email string) (result Recipient, found bool, err error) {
	rows, err := r.Db.Query("select id, email, status from recipients where email = ?", email)

	if err != nil {
		err = fmt.Errorf("couldn't get row: %v", err)
		return
	}

	defer rows.Close()

	if rows.Next() {
		found = true
		result, err = mapRow(rows)
		if err != nil {
			err = fmt.Errorf("error retrieving row: %v", err)
			return
		}
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("error iterating rows: %v", err)
	}

	return result, found, err
}

func (r *DBRepository) InsertRecipient(recipient Recipient) error {
	_, err := r.Db.Exec("insert into recipients (email, status) values (?, ?)",
		recipient.Email, RecipientStatuses.Get("new"))
	if err != nil {
		err = fmt.Errorf("couldn't perform insert: %v", err)
	}
	return err
}

func (r *DBRepository) UpdateRecipient(recipient Recipient) error {
	_, err := r.Db.Exec("UPDATE recipients SET email=?, status=? WHERE id=?",
		recipient.Email, recipient.Status, recipient.ID)
	if err != nil {
		err = fmt.Errorf("couldn't perform update: %v", err)
	}
	return err
}

func (r *DBRepository) Close() error {
	return r.Db.Close()
}

func mapRow(rows *sql.Rows) (Recipient, error) {
	var (
		id     int
		email  string
		status string
		//welcomeTime time.Time

		r Recipient
	)

	err := rows.Scan(&id, &email, &status /*&welcomeTime*/)

	if err == nil {
		r = Recipient{ID: id, Email: email, Status: RecipientStatuses.Get(status) /*WelcomeTime: welcomeTime*/}
	}

	return r, err
}

func NewRepository(dsn string) (*DBRepository, error) {
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		return nil, fmt.Errorf("couldn't open connection to %q: %v", dsn, err)
	}

	return &DBRepository{Db: db}, err
}
