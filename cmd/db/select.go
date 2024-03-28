package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"log"
	"reflect"
	"strings"
	"time"
)

// Selection statements
const (
	// selectAuditBySubmissionID statement for Audit
	selectAuditBySubmissionID = `
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

	// selectAuditByID statement for Audit
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

	// selectAuditorByID statement for Auditor
	selectAuditorByID = `
	SELECT
		auditor_id,
		username,
		role,
		audit_count
	FROM auditors WHERE auditor_id = ?;
	`

	// selectAuditsByAuditor statement for Audit
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

	// selectSubmissionByID statement for Submission
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

	// selectTicketByID statement for Ticket
	selectTicketByID = `
	SELECT
		ticket_id,
		subject,
		date_opened,
		status,
		labels,
		priority,
		closed,
		responses,
		submissions_ids,
		auditor_id,
		involved
	FROM tickets WHERE ticket_id = ?;
	`

	// selectTicketsByAuditor statement for Ticket
	selectTicketsByAuditor = `SELECT * FROM tickets WHERE auditor_id = ?;`
	// selectTicketsByStatus statement for Ticket
	selectTicketsByStatus = `SELECT * FROM tickets WHERE status = ?;`
	// selectTicketsByLabel statement for Ticket
	selectTicketsByLabel = `SELECT * FROM tickets WHERE CAST(labels as TEXT) LIKE ?;`
	// selectTicketsByPriority statement for Ticket
	selectTicketsByPriority = `SELECT * FROM tickets WHERE priority = ?;`
	// selectOpenTickets statement for Ticket
	selectOpenTickets = `SELECT * FROM tickets WHERE closed = false;`
	// selectClosedTickets statement for Ticket
	selectClosedTickets = `SELECT * FROM tickets WHERE closed = true;`

	// selectAudits statement for Audit
	selectAudits = `SELECT audit_id, submission_id FROM audits`

	// selectSIDsFromUserID statement for SIDHash
	selectSIDsFromUserID = `SELECT user_id, username, sid_hash FROM sids WHERE user_id = ?;`

	// selectUsernameFromSID statement for SIDHash
	selectUsernameFromSID = `SELECT username FROM sids WHERE sid_hash = ?;`

	// isAnAuditor statement for Auditor
	isAnAuditor = `SELECT EXISTS(SELECT 1 FROM auditors WHERE auditor_id = ?);`

	// selectRole statement for Auditor
	selectRole = `SELECT role FROM auditors WHERE auditor_id = ?;`

	// selectModels statement for ModelHashes
	selectModels = `SELECT hash, models FROM models;`
)

func (db Sqlite) GetAuditBySubmissionID(submissionID int64) (Audit, error) {
	var audit Audit
	var flags string

	err := db.QueryRowContext(db.context, selectAuditBySubmissionID, submissionID).Scan(
		&audit.id, &audit.AuditorID,
		&audit.SubmissionID, &audit.SubmissionUsername, &audit.SubmissionUserID,
		&flags, &audit.ActionTaken,
	)
	if err != nil {
		return audit, err
	}

	for _, flag := range strings.Split(flags, ",") {
		audit.Flags = append(audit.Flags, Flag(flag))
	}

	if audit.AuditorID == nil {
		return audit, errors.New("error: auditor id cannot be nil")
	}

	audit.auditor, err = db.GetAuditorByID(*audit.AuditorID)
	if err != nil {
		return audit, err
	}

	return audit, nil
}

func (db Sqlite) GetAuditByID(auditID int64) (Audit, error) {
	var audit Audit
	var flags string

	err := db.QueryRowContext(db.context, selectAuditByID, auditID).Scan(
		&audit.id, &audit.AuditorID,
		&audit.SubmissionID, &audit.SubmissionUsername, &audit.SubmissionUserID,
		&flags, &audit.ActionTaken,
	)
	if err != nil {
		return audit, err
	}

	for _, flag := range strings.Split(flags, ",") {
		audit.Flags = append(audit.Flags, Flag(flag))
	}

	if audit.AuditorID == nil {
		return audit, errors.New("error: auditor id cannot be nil")
	}

	audit.auditor, err = db.GetAuditorByID(*audit.AuditorID)
	if err != nil {
		return audit, err
	}

	return audit, nil
}

func (db Sqlite) GetAuditsByAuditor(auditorID int64) ([]Audit, error) {
	auditor, err := db.GetAuditorByID(auditorID)
	if err != nil {
		return nil, fmt.Errorf("got an error while getting auditor by id: %w", err)
	}

	rows, err := db.QueryContext(db.context, selectAuditsByAuditor, auditorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var audits []Audit
	for rows.Next() {
		var audit = Audit{
			auditor: auditor,
		}

		var flags string
		err = rows.Scan(
			&audit.id, &audit.AuditorID,
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

func (db Sqlite) GetAuditorByID(auditorID int64) (Auditor, error) {
	var auditor Auditor
	var role string

	err := db.QueryRowContext(db.context, selectAuditorByID, auditorID).Scan(
		&auditor.UserID, &auditor.Username, &role, &auditor.AuditCount,
	)
	if err != nil {
		return auditor, err
	}

	auditor.Role = RoleLevel(role)

	return auditor, nil
}

func (db Sqlite) GetSubmissionByID(submissionID int64) (Submission, error) {
	var submission Submission
	var timeString string
	var fileID sql.NullString
	var ratings []byte
	var keywords []byte

	err := db.QueryRowContext(db.context, selectSubmissionByID, submissionID).Scan(
		&submission.ID, &submission.UserID, &submission.URL, &submission.AuditID,
		&submission.Title, &submission.Description, &timeString,
		&submission.Generated, &submission.Assisted, &submission.Img2Img, &ratings,
		&keywords, &fileID,
	)
	if err != nil {
		return submission, err
	}

	submission.Updated, err = time.Parse(time.RFC3339Nano, timeString)
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

	if submission.AuditID != nil {
		if *submission.AuditID == 0 {
			return submission, errors.New("error: audit ID cannot be 0")
		}
		audit, err := db.GetAuditByID(*submission.AuditID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return submission, err
			} else {
				log.Printf("warning: audit %d is not null but couldn't find audit of %d", *submission.AuditID, submissionID)
			}
		} else {
			submission.audit = &audit
			submission.AuditID = &audit.id
		}
	} else {
		// Try to get the audit by submission id
		audit, err := db.GetAuditBySubmissionID(submission.ID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return submission, err
			}
		} else {
			submission.audit = &audit
			submission.AuditID = &audit.id
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

func (db Sqlite) GetTicketByID(ticketID int64) (Ticket, error) {
	var ticket Ticket
	var dateOpened string
	var labels []byte
	var responses []byte
	var submissionIDs []byte
	var involved []byte

	err := db.QueryRowContext(db.context, selectTicketByID, ticketID).Scan(
		&ticket.ID, &ticket.Subject, &dateOpened,
		&ticket.Status, &labels, &ticket.Priority, &ticket.Closed,
		&responses, &submissionIDs, &ticket.AssignedID, &involved,
	)
	if err != nil {
		return ticket, err
	}

	err = scan(map[any]any{
		&ticket.DateOpened:    dateOpened,
		&ticket.Labels:        labels,
		&ticket.Responses:     responses,
		&ticket.SubmissionIDs: submissionIDs,
		&ticket.UsersInvolved: involved,
	})

	return ticket, nil
}

// GetTicketsByAuditor returns a list of tickets by auditor id
func (db Sqlite) GetTicketsByAuditor(auditorID int64) ([]Ticket, error) {
	return db.ticketsByQuery(selectTicketsByAuditor, auditorID)
}
func (db Sqlite) GetTicketsByStatus(status string) ([]Ticket, error) {
	return db.ticketsByQuery(selectTicketsByStatus, status)
}

// GetTicketsByLabel returns a slice of Ticket by label using selectTicketsByLabel
func (db Sqlite) GetTicketsByLabel(label string) ([]Ticket, error) {
	label = strings.ReplaceAll(label, " ", "_")
	return db.ticketsByQuery(selectTicketsByLabel, "%"+label+"%")
}
func (db Sqlite) GetTicketsByPriority(priority string) ([]Ticket, error) {
	return db.ticketsByQuery(selectTicketsByPriority, priority)
}
func (db Sqlite) GetOpenTickets() ([]Ticket, error) {
	return db.ticketsByQuery(selectOpenTickets)
}
func (db Sqlite) GetClosedTickets() ([]Ticket, error) {
	return db.ticketsByQuery(selectClosedTickets)
}

func (db Sqlite) ticketsByQuery(query string, args ...any) ([]Ticket, error) {
	rows, err := db.QueryContext(db.context, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error: querying tickets by auditor: %w", err)
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var ticket Ticket
		var dateOpened string
		var labels []byte
		var responses []byte
		var submissionIDs []byte
		var involved []byte

		err := rows.Scan(
			&ticket.ID, &ticket.Subject, &dateOpened,
			&ticket.Status, &labels, &ticket.Priority, &ticket.Closed,
			&responses, &submissionIDs, &ticket.AssignedID, &involved,
		)
		if err != nil {
			return nil, err
		}

		err = scan(map[any]any{
			&ticket.DateOpened:    dateOpened,
			&ticket.Labels:        labels,
			&ticket.Responses:     responses,
			&ticket.SubmissionIDs: submissionIDs,
			&ticket.UsersInvolved: involved,
		})
		if err != nil {
			return nil, err
		}

		tickets = append(tickets, ticket)
	}

	if len(tickets) == 0 {
		return nil, fmt.Errorf("finished querying tickets by auditor: %w", sql.ErrNoRows)
	}

	return tickets, nil
}

// scan scans the map of addresses to values and sets the values to the addresses
func scan(scan map[any]any) error {
	for key, value := range scan {
		k := reflect.ValueOf(key)
		if k.Kind() != reflect.Pointer {
			return fmt.Errorf("error: key %T is not a pointer", key)
		}
		e := k.Elem()
		if !e.CanSet() {
			return fmt.Errorf("error: key %T cannot be set", key)
		}
		if e.Type().AssignableTo(reflect.TypeFor[time.Time]()) {
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("error: value %v is not a string", value)
			}
			p, err := time.Parse(time.RFC3339Nano, s)
			if err != nil {
				return err
			}
			e.Set(reflect.ValueOf(p))
			continue
		}
		data, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("error: value %v is not []byte for JSON unmarshalling", value)
		}
		if err := json.Unmarshal(data, k.Elem().Addr().Interface()); err != nil {
			return fmt.Errorf("error unmarshalling JSON: %w", err)
		}
	}

	return nil
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

	err = json.Unmarshal(hashes, &sid.hashes)

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
