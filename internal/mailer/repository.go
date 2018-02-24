package mailer

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Repository interface {
	GetUsersNotSubscribed() ([]User, error)
	GetUserByEmail(email string) (User, bool, error)
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

var UserStatuses = userStatusSet{"new", "subscribed"}

func (r *DBRepository) GetUsersNotSubscribed() ([]User, error) {
	var result []User
	err := r.doInTx(false, func(tx *sql.Tx) error {
		rows, err := tx.Query("select id, email, status from users where status != ?",
			UserStatuses.Get("subscribed"))

		defer rows.Close()

		if err != nil {
			return fmt.Errorf("couldn't get row: %v", err)
		}
		for rows.Next() {
			user, err := mapRow(rows)
			if err != nil {
				return fmt.Errorf("error retrieving row: %v", err)
			}
			result = append(result, user)
		}
		if err = rows.Err(); err != nil {
			return fmt.Errorf("error iterating rows: %v", err)
		}
		return nil
	})
	return result, err
}

func (r *DBRepository) GetUserByEmail(email string) (User, bool, error) {
	var result User
	var found bool
	err := r.doInTx(false, func(tx *sql.Tx) error {
		rows, err := tx.Query("select id, email, status from users where email = ?",
			email)

		defer rows.Close()

		if err != nil {
			return fmt.Errorf("couldn't get row: %v", err)
		}
		if rows.Next() {
			found = true
			result, err = mapRow(rows)
			if err != nil {
				return fmt.Errorf("error retrieving row: %v", err)
			}
		}
		if err = rows.Err(); err != nil {
			return fmt.Errorf("error iterating rows: %v", err)
		}
		return nil
	})
	return result, found, err
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
		_, err := tx.Exec("UPDATE users SET email=?, status=? WHERE id=?", user.Email,
			UserStatuses.Get("new"), user.ID)
		if err != nil {
			return fmt.Errorf("couldn't perform update: %v", err)
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
		if err != nil {
			err = fmt.Errorf("couldn't commit tx: %v", err)
		}
	}

	return err
}

func mapRow(rows *sql.Rows) (User, error) {
	var (
		id     int
		email  string
		status string
		//welcomeTime time.Time

		user User
	)

	err := rows.Scan(&id, &email, &status /*&welcomeTime*/)

	if err == nil {
		user = User{ID: id, Email: email, Status: UserStatuses.Get(status) /*WelcomeTime: welcomeTime*/}
	}

	return user, err
}

func NewRepository(dsn string) (*DBRepository, error) {
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		return nil, fmt.Errorf("couldn't open connection to %q: %v", dsn, err)
	}

	return &DBRepository{Db: db}, err
}
