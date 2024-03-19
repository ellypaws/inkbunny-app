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
)

const (
	dbFile              string = "audits.sqlite"
	getCurrentMigration string = `PRAGMA user_version;`
	setCurrentMigration string = `PRAGMA user_version = ?;`
	setForeignKeyCheck  string = `PRAGMA foreign_keys = ON;`
)

// sql statements
const (
	createAuditors = `
	CREATE TABLE IF NOT EXISTS auditors (
		id INTEGER PRIMARY KEY,
		username TEXT NOT NULL,
		role TEXT NOT NULL,
		audit_count INTEGER NOT NULL
	)
	`

	createAudits = `
	CREATE TABLE IF NOT EXISTS audits (
	    		id INTEGER PRIMARY KEY AUTOINCREMENT,
-- 	    		get auditor from auditors table, store only the auditor id
	    		auditor INTEGER,
	    		submission_id TEXT NOT NULL UNIQUE,
	    		submission_username TEXT NOT NULL,
	    		submission_user_id TEXT NOT NULL,
	    		flags TEXT NOT NULL,
	    		action_taken TEXT NOT NULL,
	    		FOREIGN KEY(auditor) REFERENCES auditors(id)
	)
	`

	createFiles = `
	CREATE TABLE IF NOT EXISTS files (
	    id INTEGER PRIMARY KEY,
-- 	    store file as a json string
	    file TEXT NOT NULL,
-- 	    store info as a json string
	    info TEXT
--		blob is not stored in the db (for now)
	)
	`

	createSubmissions = `
	CREATE TABLE IF NOT EXISTS submissions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		url TEXT NOT NULL,
-- 		get audit from audits table, store only the audit id
		audit INTEGER UNIQUE,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		ai_generated BOOLEAN NOT NULL,
		ai_assisted BOOLEAN NOT NULL,
		img2img BOOLEAN NOT NULL,
		ratings TEXT NOT NULL,
-- 		store keywords as a json string
		keywords TEXT,
-- 		get files from files table, store only the file ids
		files INTEGER,
	    FOREIGN KEY(audit) REFERENCES audits(id),
	    FOREIGN KEY(files) REFERENCES files(id)
	)
	`
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
	{migrationName: "create audits table", migrationQuery: createAudits},
	{migrationName: "create files table", migrationQuery: createFiles},
	{migrationName: "create submissions table", migrationQuery: createSubmissions},
}

func New(ctx context.Context) (*Sqlite, error) {
	filename, err := DBFilename()
	if err != nil {
		return nil, err
	}

	err = touchDBFile(filename)
	if err != nil {
		return nil, errors.New("failed to create db file")
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