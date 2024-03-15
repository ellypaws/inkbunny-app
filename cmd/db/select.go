package db

// Selection statements
const (
	selectAuditsBySubmission = `
	SELECT 
	    auditor,
	    submission_username,
	    submission_user_id,
	    flags, submission_id,
	    action_taken
	FROM audits WHERE submission_id = ?;
	`

	selectAuditorByID = `
	SELECT id,
	       username,
	       role,
	       audit_count
	FROM auditors WHERE id = ?;
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
