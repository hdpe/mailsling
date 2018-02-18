package mailer

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Repository interface {
	GetUsersNotWelcomed() ([]User, error)
	InsertUser(user User) error
	UpdateUser(user User) error
}

type DBRepository struct {
	Db *sql.DB
}

type User struct {
	ID          int
	Email       string
	Status      UserStatus
	WelcomeTime time.Time
}

type UserStatus string
type userStatusSet []UserStatus

func (r userStatusSet) Get(name string) UserStatus {
	for _, us := range r {
		if string(us) == name {
			return us
		}
	}
	panic(fmt.Sprintf("Unknown user status %q", name))
}

var UserStatuses = userStatusSet{"new", "welcomed"}

func (r *DBRepository) GetUsersNotWelcomed() ([]User, error) {
	var result []User
	err := r.doInTx(false, func(tx *sql.Tx) error {
		rows, err := tx.Query("select id, email, status, welcome_time from users")
		if err != nil {
			return fmt.Errorf("couldn't perform insert: %v", err)
		}
		for rows.Next() {
			var (
				id          *int
				email       *string
				status      *string
				welcomeTime *time.Time
			)

			rows.Scan(email)
			result = append(result, User{ID: *id, Email: *email, Status: UserStatuses.Get(*status), WelcomeTime: *welcomeTime})
		}
		return nil
	})
	return result, err
}

func (r *DBRepository) InsertUser(user User) error {
	return r.doInTx(false, func(tx *sql.Tx) error {
		_, err := tx.Exec("insert into users (email, status) values (?, ?)", user.Email,
			UserStatuses.Get("new"))
		if err != nil {
			return fmt.Errorf("couldn't perform insert: %v", err)
		} else {
			return nil
		}
	})
}

func (r *DBRepository) UpdateUser(user User) error {
	return r.doInTx(false, func(tx *sql.Tx) error {
		_, err := tx.Exec("UPDATE users SET email=?, status=?, welcome_time=? WHERE id=?", user.Email,
			UserStatuses.Get("new"), user.WelcomeTime, user.ID)
		if err != nil {
			return fmt.Errorf("couldn't perform insert: %v", err)
		} else {
			return nil
		}
	})
}

func (r *DBRepository) doInTx(readOnly bool, action func(tx *sql.Tx) error) error {
	tx, err := r.Db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: readOnly})

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err != nil {
		return fmt.Errorf("couldn't start tx: %v", err)
	}

	err = action(tx)

	if readOnly || err != nil {
		tx.Rollback()
	} else {
		err = tx.Commit()
		fmt.Errorf("couldn't commit tx: %v", err)
	}

	return err
}

func NewRepository(dsn string) (*DBRepository, error) {
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		return nil, fmt.Errorf("couldn't open connection to %q: %v", dsn, err)
	}

	return &DBRepository{Db: db}, err
}
