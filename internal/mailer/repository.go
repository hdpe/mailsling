package mailer

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/hdpe/mailsling/internal/mailer/schema"
	"github.com/mattes/migrate"
	"github.com/mattes/migrate/database/mysql"

	"github.com/mattes/migrate/source/go-bindata"
)

type listRecipientComposite struct {
	listRecipientID int
	recipientID     int
	email           string
	listID          string
	status          RecipientStatus
}

type Repository interface {
	GetRecipientDataByStatus(*sql.Tx, []RecipientStatus) ([]listRecipientComposite, error)
	GetRecipientByEmail(*sql.Tx, string) (recipient Recipient, found bool, err error)
	InsertRecipient(*sql.Tx, Recipient) (int, error)
	GetListRecipient(*sql.Tx, int) (ListRecipient, error)
	GetListRecipientByEmailAndListID(tx *sql.Tx, email string, listID string) (listRecipient ListRecipient, found bool, err error)
	InsertListRecipient(*sql.Tx, ListRecipient) (int, error)
	UpdateListRecipient(*sql.Tx, ListRecipient) error
	DoInTx(func(*sql.Tx) error) error
	Close() error
}

type DBRepository struct {
	Db *sql.DB
}

func (r *DBRepository) GetRecipientDataByStatus(tx *sql.Tx, statuses []RecipientStatus) (result []listRecipientComposite, err error) {
	rows, err := tx.Query(fmt.Sprintf(`
		select lr.id, r.id, r.email, lr.list_id, lr.status 
		from recipients r 
			inner join list_recipients lr
				on r.id = lr.recipient_id
		where %v`, toStatusInFragment(statuses)))

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

func toStatusInFragment(statuses []RecipientStatus) string {
	strs := make([]string, len(statuses))
	for i, s := range statuses {
		strs[i] = fmt.Sprintf("'%s'", s)
	}

	return fmt.Sprintf("status in (%s)", strings.Join(strs, ", "))
}

func (r *DBRepository) GetListRecipient(tx *sql.Tx, id int) (lr ListRecipient, err error) {
	lr, err = r.getListRecipientInternal(tx, id)
	if err != nil {
		return
	}

	attribs, err := r.getListRecipientAttributes(tx, id)
	if err == nil {
		lr.attribs = attribs
	}

	return
}

func (r *DBRepository) getListRecipientInternal(tx *sql.Tx, id int) (result ListRecipient, err error) {
	rows, err := tx.Query("select id, list_id, recipient_id, status, last_modified from list_recipients where id = ?", id)

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
		return
	}

	return result, err
}

func (r *DBRepository) GetRecipientByEmail(tx *sql.Tx, email string) (result Recipient, found bool, err error) {
	rows, err := tx.Query("select id, email from recipients where email = ?", email)

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

func (r *DBRepository) InsertRecipient(tx *sql.Tx, recipient Recipient) (int, error) {
	res, err := tx.Exec("insert into recipients (email) values (?)", recipient.Email)
	if err != nil {
		return 0, fmt.Errorf("couldn't perform insert: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("couldn't get inserted row ID: %v", err)
	}
	return int(id), nil
}

func (r *DBRepository) GetListRecipientByEmailAndListID(tx *sql.Tx, email string, listID string) (
	lr ListRecipient, found bool, err error) {
	lr, found, err = r.getListRecipientByEmailAndListIDInternal(tx, email, listID)
	if !found || err != nil {
		return
	}

	attribs, err := r.getListRecipientAttributes(tx, lr.id)
	if err == nil {
		lr.attribs = attribs
	}

	return
}

func (r *DBRepository) getListRecipientByEmailAndListIDInternal(tx *sql.Tx, email string, listID string) (
	result ListRecipient, found bool, err error) {
	rows, err := tx.Query(`
		select lr.id, lr.list_id, lr.recipient_id, lr.status, lr.last_modified
		from list_recipients lr
			inner join recipients r 
				on lr.recipient_id = r.id
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
		return
	}

	return result, found, err
}

func (r *DBRepository) InsertListRecipient(tx *sql.Tx, listRecipient ListRecipient) (int, error) {
	res, err := tx.Exec("insert into list_recipients (list_id, recipient_id, status, last_modified) values (?, ?, ?, ?)",
		listRecipient.listID, listRecipient.recipientID, RecipientStatuses.Get("new"), listRecipient.lastModified)
	if err != nil {
		return 0, fmt.Errorf("couldn't perform insert: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("couldn't get inserted row ID: %v", err)
	}
	err = r.updateListRecipientAttributes(tx, int(id), listRecipient.attribs)
	return int(id), err
}

func (r *DBRepository) UpdateListRecipient(tx *sql.Tx, listRecipient ListRecipient) error {
	_, err := tx.Exec("update list_recipients set status = ?, last_modified = ? where id = ?",
		listRecipient.status, listRecipient.lastModified, listRecipient.id)
	if err != nil {
		return fmt.Errorf("couldn't perform update: %v", err)
	}
	err = r.updateListRecipientAttributes(tx, listRecipient.id, listRecipient.attribs)
	return err
}

func (r *DBRepository) updateListRecipientAttributes(tx *sql.Tx, listRecipientID int, attribs map[string]string) error {
	_, err := tx.Exec("delete from list_recipient_attributes where list_recipient_id = ?", listRecipientID)
	if err != nil {
		return fmt.Errorf("couldn't delete existing attributes: %v", err)
	}
	for k, v := range attribs {
		_, err := tx.Exec("insert into list_recipient_attributes (list_recipient_id, `key`, `value`) values (?, ?, ?)",
			listRecipientID, k, v)
		if err != nil {
			return fmt.Errorf("couldn't insert attribute: %v", err)
		}
	}
	return nil
}

func (r *DBRepository) getListRecipientAttributes(tx *sql.Tx, listRecipientID int) (result map[string]string, err error) {
	result = make(map[string]string)
	rows, err := tx.Query("select `key`, `value` from list_recipient_attributes where list_recipient_id = ?", listRecipientID)
	if err != nil {
		err = fmt.Errorf("couldn't get row: %v", err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var (
			key   string
			value string
		)
		err = rows.Scan(&key, &value)
		if err != nil {
			err = fmt.Errorf("error retrieving row: %v", err)
			return
		}
		result[key] = value
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("error iterating rows: %v", err)
	}

	return result, err
}

func (r *DBRepository) DoInTx(action func(tx *sql.Tx) error) error {
	tx, err := r.Db.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				err = fmt.Errorf("error rolling back tx: %v after %v", rollbackErr, err)
			}
		} else {
			err = tx.Commit()
		}
	}()

	err = action(tx)

	return err
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
		id           int
		listID       string
		recipientID  int
		status       string
		lastModified time.Time

		r ListRecipient
	)

	err := rows.Scan(&id, &listID, &recipientID, &status, &lastModified)

	if err == nil {
		r = ListRecipient{id: id, listID: listID, recipientID: recipientID, status: RecipientStatuses.Get(status), lastModified: lastModified}
	}

	return r, err
}

func mapListRecipientCompositeRow(rows *sql.Rows) (listRecipientComposite, error) {
	var (
		listRecipientID int
		recipientID     int
		email           string
		listID          string
		status          string

		r listRecipientComposite
	)

	err := rows.Scan(&listRecipientID, &recipientID, &email, &listID, &status)

	if err == nil {
		r = listRecipientComposite{
			listRecipientID: listRecipientID,
			recipientID:     recipientID,
			email:           email,
			listID:          listID,
			status:          RecipientStatuses.Get(status),
		}
	}

	return r, err
}

func NewRepository(dsn string) (*DBRepository, error) {
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		return nil, fmt.Errorf("couldn't open connection to %q: %v", dsn, err)
	}

	err = applyMigrations(db)

	if err != nil {
		return nil, fmt.Errorf("couldn't apply migrations: %v", err)
	}

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
