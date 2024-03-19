package db

import (
	"fmt"
	"log"
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
	                         ratings, keywords, files)
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
	                  files=excluded.files;
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

func (db Sqlite) IncreaseAuditCount(auditorID string) error {
	_, err := db.ExecContext(db.context, increaseAuditCount, auditorID)
	return err
}

func (db Sqlite) SyncAuditCount(auditorID string) error {
	audits, err := db.GetAuditsByAuditor(auditorID)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(db.context, updateAuditCount, len(audits), auditorID)
	return err
}

func (db Sqlite) InsertAudit(audit Audit) (id int64, err error) {
	res, err := db.ExecContext(db.context, upsertAudit,
		audit.Auditor.UserID,
		audit.SubmissionID, audit.SubmissionUsername, audit.SubmissionUserID,
		fmt.Sprintf("%v", audit.Flags), audit.ActionTaken,
	)
	if err != nil {
		return 0, err
	}

	id, err = res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// set audit in submission if it exists in database
	_, err = db.ExecContext(db.context, updateSubmissionAudit, id, audit.SubmissionID)
	if err != nil {
		return
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		log.Printf("error: getting rows affected: %v", err)
		return
	}

	if rowCount == 0 {
		log.Printf("warning: submission %s doesn't exist in the database", audit.SubmissionID)
	}

	return id, nil
}

func (db Sqlite) InsertFile(file File) error {
	_, err := db.ExecContext(db.context, upsertFile,
		file.File.FileID, file.File, file.Info,
	)

	return err
}

func (db Sqlite) InsertSubmission(submission Submission) error {
	var auditId *int64
	if submission.Audit != nil {
		id, err := db.InsertAudit(*submission.Audit)
		if err != nil {
			return err
		}
		auditId = &id
	}

	_, err := db.ExecContext(db.context, upsertSubmission,
		submission.ID, submission.UserID, submission.URL, auditId,
		submission.Title, submission.Description, submission.Updated,
		submission.Generated, submission.Assisted, submission.Img2Img,
		submission.Ratings, submission.Keywords, submission.Files,
	)

	return err
}
