package db

// use sqlite https://modernc.org/sqlite/

import (
	"context"
	"database/sql"
	"github.com/go-errors/errors"
	"log"
	_ "modernc.org/sqlite"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	dbFile              string = "audits.sqlite"
	getCurrentMigration string = `PRAGMA user_version;`
	setCurrentMigration string = `PRAGMA user_version = ?;`
	setForeignKeyCheck  string = `PRAGMA foreign_keys = ON;`
)

type Sqlite struct {
	*sql.DB
	context context.Context
}

type migration struct {
	migrationName  string
	migrationQuery string
}

var migrations = []migration{
	{migrationName: "create auditors table", migrationQuery: createAuditors},
	{migrationName: "create submissions table", migrationQuery: createSubmissions},
	{migrationName: "create audits table", migrationQuery: createAudits},
	{migrationName: "create tickets table", migrationQuery: createTickets},
	{migrationName: "create sids table", migrationQuery: createSIDs},
	{migrationName: "create models table", migrationQuery: createModels},
}

// sql statements
const (
	// createAuditors statement for Auditor
	createAuditors = `
	CREATE TABLE IF NOT EXISTS auditors (
		auditor_id INTEGER PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		role TEXT NOT NULL DEFAULT 'user',
		audit_count INTEGER NOT NULL DEFAULT 0
	)
	`

	// createSubmissions statement for Submission
	createSubmissions = `
	CREATE TABLE IF NOT EXISTS submissions (
		submission_id INTEGER PRIMARY KEY,
		user_id TEXT NOT NULL,
		url TEXT NOT NULL,
--		get audit from audits table, store only the audit id
		audit_id INTEGER UNIQUE,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		metadata BLOB,
		ratings BLOB,
--		store keywords as a json string
		keywords BLOB,
--		get files from files table, store only the file ids
		files BLOB
--		FOREIGN KEY(file) REFERENCES files(file)
	)
	`

	// createAudits statement for Audit
	createAudits = `
	CREATE TABLE IF NOT EXISTS audits (
				audit_id INTEGER PRIMARY KEY AUTOINCREMENT,
--				get auditor from auditors table, store only the auditor id
				auditor_id INTEGER,
				submission_id TEXT NOT NULL UNIQUE,
				submission_username TEXT NOT NULL,
				submission_user_id TEXT NOT NULL,
				flags TEXT NOT NULL,
				action_taken TEXT NOT NULL,
				FOREIGN KEY(auditor_id) REFERENCES auditors(auditor_id),
				FOREIGN KEY(submission_id) REFERENCES submissions(submission_id)
	)
	`

	// createTickets statement for Ticket
	createTickets = `
	CREATE TABLE IF NOT EXISTS tickets (
				ticket_id INTEGER PRIMARY KEY AUTOINCREMENT,
				subject TEXT NOT NULL,
				date_opened TEXT NOT NULL,
				date_closed TEXT,
				status TEXT NOT NULL DEFAULT 'Open',
				labels BLOB,
				priority TEXT NOT NULL DEFAULT 'Low',
				flags BLOB,
				closed BOOLEAN NOT NULL DEFAULT FALSE,
				responses BLOB NOT NULL,
				submissions_ids BLOB,
				auditor_id INTEGER,
				involved BLOB,
				FOREIGN KEY(auditor_id) REFERENCES auditors(auditor_id)
	)
	`

	// createSIDs statement for SIDHash
	createSIDs = `
	CREATE TABLE IF NOT EXISTS sids (
	    		sid_hash TEXT PRIMARY KEY,
				auditor_id INTEGER NOT NULL
	)
	`

	// createModels statement for ModelHashes
	createModels = `
	CREATE TABLE IF NOT EXISTS models (
		hash TEXT PRIMARY KEY,
		models BLOB
	)
	`
)

// New creates a new Sqlite database connection
// Use context to pass in the filename of the database
//
//	context.WithValue(context.Background(), "filename", "audits.sqlite")
//
// Alternatively, use context to pass in ":memory:" to create an in-memory database
func New(ctx context.Context) (*Sqlite, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var filename string
	if ctx.Value(":memory:") != nil {
		filename = ":memory:"
	}
	if ctx.Value("filename") != nil {
		if s, ok := ctx.Value("filename").(string); ok {
			filename = s
		}
	}

	if filename == "" {
		var err error
		filename, err = DBFilename()
		if err != nil {
			return nil, err
		}

		err = touchDBFile(filename)
		if err != nil {
			return nil, errors.New("failed to create db file")
		}
	}

	db, err := sql.Open("sqlite", filename)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(setForeignKeyCheck)
	if err != nil {
		return nil, errors.New("failed to enable foreign key checks")
	}

	err = migrate(ctx, db)
	if err != nil {
		return nil, err
	}

	return &Sqlite{db, ctx}, nil
}

func (db Sqlite) Context() context.Context {
	return db.context
}

func DBFilename() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return dir + "/" + dbFile, nil
}

func touchDBFile(filename string) error {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		file, createErr := os.Create(filename)
		if createErr != nil {
			return createErr
		}

		closeErr := file.Close()
		if closeErr != nil {
			return closeErr
		}
	}

	return nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	var currentMigration int

	row := db.QueryRowContext(ctx, getCurrentMigration)

	err := row.Scan(&currentMigration)
	if err != nil {
		return err
	}

	requiredMigration := len(migrations)

	log.Printf("Current DB version: %v, required DB version: %v\n", currentMigration, requiredMigration)

	if currentMigration < requiredMigration {
		for migrationNum := currentMigration + 1; migrationNum <= requiredMigration; migrationNum++ {
			err = execMigration(ctx, db, migrationNum)
			if err != nil {
				log.Printf("Error running migration %v '%v'\n", migrationNum, migrations[migrationNum-1].migrationName)

				return err
			}
		}
	}

	return nil
}

func execMigration(ctx context.Context, db *sql.DB, migrationNum int) error {
	log.Printf("Running migration %v '%v'\n", migrationNum, migrations[migrationNum-1].migrationName)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	//nolint
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, migrations[migrationNum-1].migrationQuery)
	if err != nil {
		return err
	}

	setQuery := strings.Replace(setCurrentMigration, "?", strconv.Itoa(migrationNum), 1)

	_, err = tx.ExecContext(ctx, setQuery)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

var nilDatabase = errors.New("database error")

const timeout = 15 * time.Second

func Error(db *Sqlite) error {
	if db == nil {
		return nilDatabase
	}
	ctx, cancel := context.WithTimeout(db.Context(), timeout)
	defer cancel()
	return db.PingContext(ctx)
}
