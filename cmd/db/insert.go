package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

// Insert statements
const (
	upsertAuditor = `
	INSERT INTO auditors (auditor_id, username, role, audit_count) VALUES (?, ?, ?, ?)
	ON CONFLICT(auditor_id) DO UPDATE SET username=excluded.username, role=excluded.role, audit_count=excluded.audit_count;
	`

	increaseAuditCount = `
	UPDATE auditors SET audit_count = audit_count + 1 WHERE auditor_id = ?;
	`

	updateAuditCount = `
	UPDATE auditors SET audit_count = ? WHERE auditor_id = ?;
	`

	upsertAudit = `
	INSERT INTO audits (auditor_id,
	                    submission_id, submission_username, submission_user_id,
	                    flags, action_taken)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(submission_id)
	    DO UPDATE SET
	                  auditor_id=excluded.auditor_id,
	                  submission_username=excluded.submission_username,
	                  submission_user_id=excluded.submission_user_id,
	                  flags=excluded.flags,
	                  action_taken=excluded.action_taken;
	`

	upsertFile = `
	INSERT INTO files (file_id, file, info) VALUES (?, ?, ?)
	ON CONFLICT(file_id) DO UPDATE SET file=excluded.file, info=excluded.info;
	`

	upsertSubmission = `
--  Audit is a foreign key, but it's not required. Only give an integer if it exists.
	INSERT INTO submissions (submission_id, user_id, url, audit_id,
	                         title, description, updated_at,
	                         ai_generated, ai_assisted, img2img,
	                         ratings, keywords, file_id)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(submission_id)
	    DO UPDATE SET
	                  user_id=excluded.user_id,
	                  url=excluded.url,
	                  audit_id=excluded.audit_id,
	                  title=excluded.title,
	                  description=excluded.description,
	                  updated_at=excluded.updated_at,
	                  ai_generated=excluded.ai_generated,
	                  ai_assisted=excluded.ai_assisted,
	                  img2img=excluded.img2img,
	                  ratings=excluded.ratings,
	                  keywords=excluded.keywords,
	                  file_id=excluded.file_id;
	`

	updateSubmissionDescription = `
	UPDATE submissions SET description = ? WHERE submission_id = ?;
	`

	// IF submission exists, update the audit_id field
	updateSubmissionAudit = `
	UPDATE submissions SET audit_id = ? WHERE submission_id = ?;
	`
)

func (db Sqlite) InsertAuditor(auditor Auditor) error {
	_, err := db.ExecContext(db.context, upsertAuditor,
		auditor.UserID, auditor.Username, auditor.Role, auditor.AuditCount,
	)

	return err
}

func (db Sqlite) IncreaseAuditCount(auditorID int64) error {
	_, err := db.ExecContext(db.context, increaseAuditCount, auditorID)
	return err
}

func (db Sqlite) SyncAuditCount(auditorID int64) error {
	audits, err := db.GetAuditsByAuditor(auditorID)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(db.context, updateAuditCount, len(audits), auditorID)
	return err
}

// InsertAudit inserts an audit into the database. If the auditor is not in the database, it will be inserted.
// Auditor needs to be non-empty or exist in the database before inserting an audit.
// Similarly, the Submission needs to be in the database as well and be filled in the audit.
// If successful, the submission will be updated with the new audit_id.
func (db Sqlite) InsertAudit(audit Audit) (int64, error) {
	if audit.Auditor == nil {
		return 0, errors.New("error: auditor is nil")
	}

	if audit.Auditor != nil {
		_, err := db.GetAuditorByID(audit.Auditor.UserID)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			// Only insert if new, otherwise keep old record
			err := db.InsertAuditor(*audit.Auditor)
			if err != nil {
				return 0, fmt.Errorf("error: inserting auditor: %v", err)
			}
		}
	}

	var flags []string
	for _, flag := range audit.Flags {
		flags = append(flags, string(flag))
	}
	res, err := db.ExecContext(db.context, upsertAudit,
		audit.Auditor.UserID,
		audit.SubmissionID, audit.SubmissionUsername, audit.SubmissionUserID,
		strings.Join(flags, ","), audit.ActionTaken,
	)
	if err != nil {
		return 0, err
	}

	audit, err = db.GetAuditBySubmissionID(audit.SubmissionID)
	if err != nil {
		return 0, fmt.Errorf("error: getting audit by submission id: %v", err)
	}

	// set audit in submission if it exists in database
	res, err = db.ExecContext(db.context, updateSubmissionAudit, audit.ID, audit.SubmissionID)
	if err != nil {
		return 0, fmt.Errorf("error: updating submission audit: %v", err)
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		log.Printf("error: getting rows affected: %v", err)
		return 0, err
	}

	if rowCount == 0 {
		log.Printf("warning: submission %d doesn't exist in the database", audit.SubmissionID)
	}

	return audit.ID, nil
}

// FixAuditsInSubmissions updates all submissions with the correct audit_id.
func (db Sqlite) FixAuditsInSubmissions() error {
	rows, err := db.QueryContext(db.context, selectAudits)
	if err != nil {
		return err
	}
	defer rows.Close()

	var audits []Audit
	for rows.Next() {
		var auditID, submissionID int64
		err = rows.Scan(&auditID, &submissionID)
		if err != nil {
			return err
		}

		audits = append(audits, Audit{ID: auditID, SubmissionID: submissionID})
	}

	for _, audit := range audits {
		_, err = db.ExecContext(db.context, updateSubmissionAudit, audit.ID, audit.SubmissionID)
		if err != nil {
			return fmt.Errorf("error: updating submission audit: %v", err)
		}
	}

	return nil
}

func (db Sqlite) InsertFile(file File) error {
	marshal, err := json.Marshal(file.File)
	if err != nil {
		return fmt.Errorf("error: marshalling file: %v", err)
	}

	info, err := json.Marshal(file.Info)
	if err != nil {
		return fmt.Errorf("error: marshalling info: %v", err)
	}

	_, err = db.ExecContext(db.context, upsertFile,
		file.File.FileID, marshal, info,
	)

	return err
}

func (db Sqlite) InsertSubmission(submission Submission) error {
	ratings, err := json.Marshal(submission.Ratings)
	if err != nil {
		return fmt.Errorf("error: marshalling ratings: %v", err)
	}

	keywords, err := json.Marshal(submission.Keywords)
	if err != nil {
		return fmt.Errorf("error: marshalling keywords: %v", err)
	}

	var fileIDs []string
	if len(submission.Files) > 0 {
		for _, file := range submission.Files {
			err = db.InsertFile(file)
			if err != nil {
				return fmt.Errorf("error: inserting file: %v", err)
			}
			fileIDs = append(fileIDs, file.File.FileID)
		}
	}

	var fileIDsNullable sql.NullString
	if len(fileIDs) > 0 {
		fileIDsNullable = sql.NullString{
			String: strings.Join(fileIDs, ","),
			Valid:  true,
		}
	}

	var auditIDNullable sql.NullInt64
	var newAudit bool
	if submission.Audit != nil {
		auditIDNullable = sql.NullInt64{
			Int64: submission.Audit.ID,
			Valid: true,
		}
		newAudit = true
	} else {
		audit, err := db.GetAuditBySubmissionID(submission.ID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("error: getting audit by submission id: %v", err)
			}
		} else {
			auditIDNullable = sql.NullInt64{
				Int64: audit.ID,
				Valid: true,
			}
			submission.Audit = &audit
		}
	}

	_, err = db.ExecContext(db.context, upsertSubmission,
		submission.ID, submission.UserID, submission.URL, auditIDNullable,
		submission.Title, submission.Description, submission.Updated.UTC().Format(time.RFC3339),
		submission.Generated, submission.Assisted, submission.Img2Img,
		ratings, keywords, fileIDsNullable,
	)
	if err != nil {
		return fmt.Errorf("error: inserting submission: %v", err)
	}

	if newAudit {
		_, err := db.InsertAudit(*submission.Audit)
		if err != nil {
			return fmt.Errorf("error: inserting audit: %v", err)
		}
	}

	return nil
}

func (db Sqlite) UpdateDescription(submission Submission) error {
	_, err := db.ExecContext(db.context, updateSubmissionDescription, submission.Description, submission.ID)
	return err
}
