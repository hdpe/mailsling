package mailer

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/hdpe/mailsling/internal/mailer/schema"
	"github.com/mattes/migrate"
	"github.com/mattes/migrate/database/mysql"

	"github.com/mattes/migrate/source/go-bindata"
)

type listRecipientComposite struct {
	recipientID int
	email       string
	listID      string
	status      RecipientStatus
}

type Repository interface {
	GetNewRecipients() ([]listRecipientComposite, error)
	GetRecipientByEmail(string) (recipient Recipient, found bool, err error)
	InsertRecipient(Recipient) (int, error)
	GetListRecipient(int) (ListRecipient, error)
	GetListRecipientByEmailAndListID(email string, listID string) (listRecipient ListRecipient, found bool, err error)
	InsertListRecipient(ListRecipient) (int, error)
	UpdateListRecipient(ListRecipient) error
	DoInTx(func() error) error
	Close() error
}

type DBRepository struct {
	Db *sql.DB
}

func (r *DBRepository) GetNewRecipients() (result []listRecipientComposite, err error) {
	rows, err := r.Db.Query(`
		select r.id, r.email, lr.list_id, lr.status 
		from recipients r 
			inner join list_recipients lr
				on r.id = lr.recipient_id
		where status = ?`, RecipientStatuses.Get("new"))

	if err != nil {
		err = fmt.Errorf("couldn't get row: %v", err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var rec listRecipientComposite
		rec, err = mapListRecipientCompositeRow(rows)
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

func (r *DBRepository) GetListRecipient(id int) (result ListRecipient, err error) {
	rows, err := r.Db.Query("select id, list_id, recipient_id, status from list_recipients where id = ?", id)

	if err != nil {
		err = fmt.Errorf("couldn't get row: %v", err)
		return
	}

	defer rows.Close()

	if rows.Next() {
		result, err = mapListRecipientRow(rows)
		if err != nil {
			err = fmt.Errorf("error retrieving row: %v", err)
			return
		}
	} else {
		err = fmt.Errorf("recipient #%d not found", id)
		return
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("error iterating rows: %v", err)
	}

	return result, err
}

func (r *DBRepository) GetRecipientByEmail(email string) (result Recipient, found bool, err error) {
	rows, err := r.Db.Query("select id, email from recipients where email = ?", email)

	if err != nil {
		err = fmt.Errorf("couldn't get row: %v", err)
		return
	}

	defer rows.Close()

	if rows.Next() {
		found = true
		result, err = mapRecipientRow(rows)
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

func (r *DBRepository) InsertRecipient(recipient Recipient) (int, error) {
	res, err := r.Db.Exec("insert into recipients (email) values (?)", recipient.Email)
	if err != nil {
		return 0, fmt.Errorf("couldn't perform insert: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("couldn't get inserted row ID: %v", err)
	}
	return int(id), nil
}

func (r *DBRepository) UpdateListRecipient(recipient ListRecipient) error {
	_, err := r.Db.Exec("update list_recipients set status=? where id=?",
		recipient.status, recipient.id)
	if err != nil {
		err = fmt.Errorf("couldn't perform update: %v", err)
	}
	return err
}

func (r *DBRepository) GetListRecipientByEmailAndListID(email string, listID string) (
	result ListRecipient, found bool, err error) {
	rows, err := r.Db.Query(`
		select lr.id, lr.list_id, lr.recipient_id, lr.status
		from list_recipients lr
			inner join recipients r 
				on lr.list_id = r._id
		where r.email = ? and lr.list_id = ?`, email, listID)

	if err != nil {
		err = fmt.Errorf("couldn't get row: %v", err)
		return
	}

	defer rows.Close()

	if rows.Next() {
		found = true
		result, err = mapListRecipientRow(rows)
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

func (r *DBRepository) InsertListRecipient(listRecipient ListRecipient) (int, error) {
	res, err := r.Db.Exec("insert into list_recipients (list_id, recipient_id, status) values (?, ?, ?)",
		listRecipient.listID, listRecipient.recipientID, RecipientStatuses.Get("new"))
	if err != nil {
		return 0, fmt.Errorf("couldn't perform insert: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("couldn't get inserted row ID: %v", err)
	}
	return int(id), nil
}

func (r *DBRepository) DoInTx(action func() error) error {
	tx, err := r.Db.Begin()

	defer tx.Rollback()

	if err != nil {
		return err
	}

	err = action()

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *DBRepository) Close() error {
	return r.Db.Close()
}

func mapRecipientRow(rows *sql.Rows) (Recipient, error) {
	var (
		id    int
		email string

		r Recipient
	)

	err := rows.Scan(&id, &email)

	if err == nil {
		r = Recipient{ID: id, Email: email}
	}

	return r, err
}

func mapListRecipientRow(rows *sql.Rows) (ListRecipient, error) {
	var (
		id          int
		listID      string
		recipientID int
		status      string
		//welcomeTime time.Time

		r ListRecipient
	)

	err := rows.Scan(&id, &listID, &recipientID, &status /*&welcomeTime*/)

	if err == nil {
		r = ListRecipient{id: id, listID: listID, recipientID: recipientID, status: RecipientStatuses.Get(status) /*WelcomeTime: welcomeTime*/}
	}

	return r, err
}

func mapListRecipientCompositeRow(rows *sql.Rows) (listRecipientComposite, error) {
	var (
		recipientID int
		email       string
		listID      string
		status      string
		//welcomeTime time.Time

		r listRecipientComposite
	)

	err := rows.Scan(&recipientID, &email, &listID, &status /*&welcomeTime*/)

	if err == nil {
		r = listRecipientComposite{recipientID: recipientID, email: email, listID: listID, status: RecipientStatuses.Get(status) /*WelcomeTime: welcomeTime*/}
	}

	return r, err
}

func NewRepository(dsn string) (*DBRepository, error) {
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		return nil, fmt.Errorf("couldn't open connection to %q: %v", dsn, err)
	}

	err = applyMigrations(db)

	return &DBRepository{Db: db}, err
}

func applyMigrations(db *sql.DB) error {
	s := bindata.Resource(schema.AssetNames(),
		func(name string) ([]byte, error) {
			return schema.Asset(name)
		})

	sourceDrv, err := bindata.WithInstance(s)

	if err != nil {
		return fmt.Errorf("couldn't read migrations: %v", err)
	}

	dbDrv, err := mysql.WithInstance(db, &mysql.Config{})

	if err != nil {
		return fmt.Errorf("couldn't open connection for migrations: %v", err)
	}

	m, _ := migrate.NewWithInstance("go-bindata", sourceDrv, "mysql", dbDrv)

	migrErr := m.Up()

	if migrErr != nil && migrErr != migrate.ErrNoChange {
		err = fmt.Errorf("couldn't update database: %v", migrErr)
	}

	return err
}
