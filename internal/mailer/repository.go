package mailer

import (
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
	Close() error
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

var UserStatuses = userStatusSet{"new", "subscribed", "failed"}

func (r *DBRepository) GetUsersNotSubscribed() (result []User, err error) {
	rows, err := r.Db.Query("select id, email, status from users where status = ?",
		UserStatuses.Get("new"))

	if err != nil {
		err = fmt.Errorf("couldn't get row: %v", err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var user User
		user, err = mapRow(rows)
		if err != nil {
			err = fmt.Errorf("error retrieving row: %v", err)
			return
		}
		result = append(result, user)
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("error iterating rows: %v", err)
	}

	return result, err
}

func (r *DBRepository) GetUserByEmail(email string) (result User, found bool, err error) {
	rows, err := r.Db.Query("select id, email, status from users where email = ?", email)

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

func (r *DBRepository) InsertUser(user User) error {
	_, err := r.Db.Exec("insert into users (email, status) values (?, ?)", user.Email, UserStatuses.Get("new"))
	if err != nil {
		err = fmt.Errorf("couldn't perform insert: %v", err)
	}
	return err
}

func (r *DBRepository) UpdateUser(user User) error {
	_, err := r.Db.Exec("UPDATE users SET email=?, status=? WHERE id=?", user.Email, user.Status, user.ID)
	if err != nil {
		err = fmt.Errorf("couldn't perform update: %v", err)
	}
	return err
}

func (r *DBRepository) Close() error {
	return r.Db.Close()
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
