package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"log"
	"strings"
	"time"
)

// Selection statements
const (
	selectAuditBySubmission = `
	SELECT 
	    audit_id,
	    auditor_id,
		submission_id,
		submission_username,
		submission_user_id,
		flags,
		action_taken
	FROM audits WHERE submission_id = ?;
	`

	selectAuditByID = `
	SELECT 
	    audit_id,
	    auditor_id,
		submission_id,
		submission_username,
		submission_user_id,
		flags,
		action_taken
	FROM audits WHERE audit_id = ?;
	`

	selectAuditorByID = `
	SELECT
	    auditor_id,
	    username,
	    role,
	    audit_count
	FROM auditors WHERE auditor_id = ?;
	`

	selectAuditsByAuditor = `
	SELECT
	    audit_id,
	    auditor_id,
		submission_id,
		submission_username,
		submission_user_id,
		flags,
		action_taken
	FROM audits WHERE auditor_id = ?;
	`

	selectSubmissionByID = `
	SELECT
		submission_id,
		user_id,
		url,
		audit_id,
		title,
		description,
		updated_at,
		ai_generated,
		ai_assisted,
		img2img,
		ratings,
		keywords,
		files
	FROM submissions WHERE submission_id = ?;
	`

	selectAudits = `SELECT audit_id, submission_id FROM audits`

	selectSIDsFromUserID = `SELECT user_id, username, sid_hash FROM sids WHERE user_id = ?;`

	selectUsernameFromSID = `SELECT username FROM sids WHERE sid_hash = ?;`

	isAnAuditor = `SELECT EXISTS(SELECT 1 FROM auditors WHERE auditor_id = ?);`

	selectRole = `SELECT role FROM auditors WHERE auditor_id = ?;`

	selectModels = `SELECT hash, models FROM models;`
)

func (db Sqlite) GetAuditBySubmissionID(submissionID int64) (Audit, error) {
	var audit Audit
	var auditorID int64
	var flags string

	err := db.QueryRowContext(db.context, selectAuditBySubmission, submissionID).Scan(
		&audit.ID, &auditorID,
		&audit.SubmissionID, &audit.SubmissionUsername, &audit.SubmissionUserID,
		&flags, &audit.ActionTaken,
	)
	if err != nil {
		return audit, err
	}

	for _, flag := range strings.Split(flags, ",") {
		audit.Flags = append(audit.Flags, Flag(flag))
	}

	audit.Auditor, err = db.GetAuditorByID(auditorID)
	if err != nil {
		return audit, err
	}

	return audit, nil
}

func (db Sqlite) GetAuditByID(auditID int64) (Audit, error) {
	var audit Audit
	var auditorID int64
	var flags string

	err := db.QueryRowContext(db.context, selectAuditByID, auditID).Scan(
		&audit.ID, &auditorID,
		&audit.SubmissionID, &audit.SubmissionUsername, &audit.SubmissionUserID,
		&flags, &audit.ActionTaken,
	)
	if err != nil {
		return audit, err
	}

	for _, flag := range strings.Split(flags, ",") {
		audit.Flags = append(audit.Flags, Flag(flag))
	}

	audit.Auditor, err = db.GetAuditorByID(auditorID)
	if err != nil {
		return audit, err
	}

	return audit, nil
}

func (db Sqlite) GetAuditsByAuditor(auditorID int64) ([]Audit, error) {
	auditor, err := db.GetAuditorByID(auditorID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
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
			&audit.ID, &auditorID,
			&audit.SubmissionID, &audit.SubmissionUsername, &audit.SubmissionUserID,
			&flags, &audit.ActionTaken,
		)
		if err != nil {
			return nil, fmt.Errorf("got an error while scanning rows: %w", err)
		}

		for _, flag := range strings.Split(flags, ",") {
			audit.Flags = append(audit.Flags, Flag(flag))
		}

		audits = append(audits, audit)
	}

	return audits, nil
}

func (db Sqlite) GetAuditorByID(auditorID int64) (*Auditor, error) {
	var auditor Auditor
	var role string

	err := db.QueryRowContext(db.context, selectAuditorByID, auditorID).Scan(
		&auditor.UserID, &auditor.Username, &role, &auditor.AuditCount,
	)
	if err != nil {
		return nil, err
	}

	auditor.Role = RoleLevel(role)

	return &auditor, nil
}

func (db Sqlite) GetSubmissionByID(submissionID int64) (Submission, error) {
	var submission Submission
	var timeString string
	var auditID sql.NullInt64
	var fileID sql.NullString
	var ratings []byte
	var keywords []byte

	err := db.QueryRowContext(db.context, selectSubmissionByID, submissionID).Scan(
		&submission.ID, &submission.UserID, &submission.URL, &auditID,
		&submission.Title, &submission.Description, &timeString,
		&submission.Generated, &submission.Assisted, &submission.Img2Img, &ratings,
		&keywords, &fileID,
	)
	if err != nil {
		return submission, err
	}

	submission.Updated, err = time.Parse(time.RFC3339, timeString)
	if err != nil {
		return submission, err
	}
	if submission.Updated.IsZero() {
		submission.Updated = time.Now().UTC()
	}

	err = json.Unmarshal(ratings, &submission.Ratings)
	if err != nil {
		return submission, fmt.Errorf("error: unmarshalling ratings: %w", err)
	}
	err = json.Unmarshal(keywords, &submission.Keywords)
	if err != nil {
		return submission, fmt.Errorf("error: unmarshalling keywords: %w", err)
	}

	if auditID.Valid {
		if auditID.Int64 == 0 {
			return submission, errors.New("error: audit ID cannot be 0")
		}
		audit, err := db.GetAuditByID(auditID.Int64)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return submission, err
			} else {
				log.Printf("warning: audit %d is not null but couldn't find audit of %d", auditID.Int64, submissionID)
			}
		} else {
			submission.Audit = &audit
		}
	} else {
		// Try to get the audit by submission id
		audit, err := db.GetAuditBySubmissionID(submission.ID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return submission, err
			}
		} else {
			submission.Audit = &audit
			// Store the audit id in the submission now
			err = db.InsertSubmission(submission)
			if err != nil {
				return submission, err
			}
		}
	}

	// TODO: get files by fileID as comma separated string
	//if fileID.Valid {
	//	files, err := db.GetFilesByID(fileID.String)
	//	if err != nil {
	//		if !errors.Is(err, sql.ErrNoRows) { return submission, err }
	//	}
	//	submission.Files = files
	//}

	return submission, nil
}

func (db Sqlite) GetSIDsFromUserID(userID int64) (SIDHash, error) {
	var hashes []byte
	var sid SIDHash

	err := db.QueryRowContext(db.context, selectSIDsFromUserID, userID).Scan(
		&sid.UserID, &sid.Username, &hashes,
	)
	if err != nil {
		return sid, err
	}

	err = json.Unmarshal(hashes, &sid.Hashes)

	return sid, nil
}

func (db Sqlite) GetUsernameFromSID(sid string) (string, error) {
	var username string
	err := db.QueryRowContext(db.context, selectUsernameFromSID, sid).Scan(&username)
	return username, err
}

func (db Sqlite) IsInAuditor(auditorID int64) (bool, error) {
	var exists bool
	err := db.QueryRowContext(db.context, isAnAuditor, auditorID).Scan(&exists)
	return exists, err
}

func (db Sqlite) IsAuditorRole(auditorID int64) bool {
	role, err := db.GetRole(auditorID)
	if err != nil {
		return false
	}
	return role <= RoleAuditor
}

func (db Sqlite) GetRole(auditorID int64) (Role, error) {
	var role string
	err := db.QueryRowContext(db.context, selectRole, auditorID).Scan(&role)
	return RoleLevel(role), err
}

// GetKnownModels returns a map of model hashes to a list of known model names
func (db Sqlite) GetKnownModels() (ModelHashes, error) {
	var modelHashes ModelHashes
	var hashes []byte

	rows, err := db.QueryContext(db.context, selectModels)
	if err != nil {
		return modelHashes, err
	}
	defer rows.Close()

	for rows.Next() {
		if modelHashes == nil {
			modelHashes = make(ModelHashes)
		}
		var hash string
		err = rows.Scan(&hash, &hashes)
		if err != nil {
			return modelHashes, err
		}
		var models []string
		err = json.Unmarshal(hashes, &models)
		modelHashes[hash] = models
	}

	return modelHashes, nil
}

func (db Sqlite) ModelNamesFromHash(hash string) []string {
	models, err := db.GetKnownModels()
	if err != nil {
		return nil
	}

	if models == nil {
		return nil
	}

	if names, ok := models[hash]; ok {
		return names
	}

	return nil
}
