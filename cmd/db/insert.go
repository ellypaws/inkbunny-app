package db

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
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

	_, err = db.ExecContext(db.context, upsertSubmission,
		submission.ID, submission.UserID, submission.URL, nil,
		submission.Title, submission.Description, submission.Updated,
		submission.Generated, submission.Assisted, submission.Img2Img,
		ratings, keywords, strings.Join(fileIDs, ","),
	)
	if err != nil {
		return fmt.Errorf("error: inserting submission: %v", err)
	}

	if submission.Audit != nil {
		id, err := db.InsertAudit(*submission.Audit)
		if err != nil {
			return fmt.Errorf("error: inserting audit: %v", err)
		}

		_, err = db.ExecContext(db.context, updateSubmissionAudit, id, submission.ID)
	}

	return nil
}
