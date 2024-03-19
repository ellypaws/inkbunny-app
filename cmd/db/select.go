package db

import (
	"database/sql"
	"fmt"
	"github.com/go-errors/errors"
	"strings"
)

// Selection statements
const (
	selectAuditsBySubmission = `
	SELECT 
	    auditor_id,
	    submission_username,
	    submission_user_id,
	    flags, submission_id,
	    action_taken
	FROM audits WHERE submission_id = ?;
	`

	selectAuditorByID = `
	SELECT auditor_id,
	       username,
	       role,
	       audit_count
	FROM auditors WHERE auditor_id = ?;
	`

	selectAuditsByAuditor = `
	SELECT
		submission_id,
		submission_username,
		submission_user_id,
		flags,
		action_taken
	FROM audits WHERE auditor_id = ?;
`
)

func (db Sqlite) GetAuditBySubmissionID(submissionID string) (Audit, error) {
	var audit Audit
	var auditor string

	err := db.QueryRowContext(db.context, selectAuditsBySubmission, submissionID).Scan(
		&auditor,
		&audit.SubmissionID, &audit.SubmissionUsername, &audit.SubmissionUserID,
		&audit.Flags, &audit.ActionTaken,
	)
	if err != nil {
		return audit, err
	}

	audit.Auditor, err = db.GetAuditorByID(auditor)
	if err != nil {
		return audit, err
	}

	return audit, nil
}

func (db Sqlite) GetAuditsByAuditor(auditorID string) ([]Audit, error) {
	auditor, err := db.GetAuditorByID(auditorID)
	if err != nil && !errors.As(err, &sql.ErrNoRows) {
		return nil, fmt.Errorf("got an error while getting auditor by id (not sql.ErrNoRows): %w", err)
	}

	rows, err := db.QueryContext(db.context, selectAuditsByAuditor, auditorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var audits []Audit
	for rows.Next() {
		var audit = Audit{
			Auditor: auditor,
		}
		var flags string
		err = rows.Scan(
			&audit.SubmissionID, &audit.SubmissionUsername, &audit.SubmissionUserID,
			&flags, &audit.ActionTaken,
		)
		if err != nil {
			return nil, fmt.Errorf("got an error while scanning rows: %w", err)
		}

		for _, flag := range strings.Split(flags[1:len(flags)-1], " ") {
			audit.Flags = append(audit.Flags, Flag(flag))
		}

		audits = append(audits, audit)
	}

	return audits, nil
}

func (db Sqlite) GetAuditorByID(auditorID string) (*Auditor, error) {
	var auditor Auditor

	err := db.QueryRowContext(db.context, selectAuditorByID, auditorID).Scan(
		&auditor.UserID, &auditor.Username, &auditor.Role, &auditor.AuditCount,
	)
	if err != nil {
		return nil, err
	}

	return &auditor, nil
}
