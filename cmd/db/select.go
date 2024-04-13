package db

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ellypaws/inkbunny/api"
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

	// selectAuditorByUsername statement for Auditor
	selectAuditorByUsername = `
	SELECT
		auditor_id,
		username,
		role,
		audit_count
	FROM auditors WHERE username = ?;
	`

	// selectAllAuditors statement for Auditor
	selectAllAuditors = `SELECT * FROM auditors;`

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
		metadata,
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
		date_closed,
		status,
		labels,
		priority,
		flags,
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
	// selectALlTickets statement for Ticket
	selectAllTickets = `SELECT * FROM tickets;`

	// selectAudits statement for Audit
	selectAudits = `SELECT audit_id, submission_id FROM audits`

	// selectUsernameFromAuditorID statement to get username from Auditor table
	selectUsernameFromAuditorID = `SELECT username FROM auditors WHERE auditor_id = ?;`
	// selectAuditorIDFromUsername statement to get auditor id from Auditor table
	selectAuditorIDFromUsername = `SELECT auditor_id FROM auditors WHERE username = ?;`
	// selectSIDsFromAuditorID statement for SIDHash
	selectSIDsFromAuditorID = `SELECT sid_hash, auditor_id FROM sids WHERE auditor_id = ?;`
	// selectSIDsFromHash statement for SIDHash
	selectSIDsFromHash = `SELECT sid_hash, auditor_id FROM sids WHERE sid_hash = ?;`
	// SelectAuditorIDFromHash statement for SIDHash
	SelectAuditorIDFromHash = `SELECT auditor_id FROM sids WHERE sid_hash = ?;`

	// isAnAuditor statement for Auditor
	isAnAuditor = `SELECT EXISTS(SELECT 1 FROM auditors WHERE auditor_id = ?);`

	// selectRole statement for Auditor
	selectRole = `SELECT role FROM auditors WHERE auditor_id = ?;`

	// selectModels statement for ModelHashes
	selectModels = `SELECT hash, models FROM models;`

	// selectArtists statement for ArtistHashes
	selectArtists = `SELECT username, user_id FROM artists;`
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
	var metadata []byte
	var ratings []byte
	var keywords []byte

	err := db.QueryRowContext(db.context, selectSubmissionByID, submissionID).Scan(
		&submission.ID, &submission.UserID, &submission.URL, &submission.AuditID,
		&submission.Title, &submission.Description, &timeString,
		&metadata, &ratings, &keywords, &fileID,
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

	if metadata != nil {
		err = json.Unmarshal(metadata, &submission.Metadata)
		if err != nil {
			return submission, fmt.Errorf("error: unmarshalling metadata: %w", err)
		}
	}
	if ratings != nil {
		err = json.Unmarshal(ratings, &submission.Ratings)
		if err != nil {
			return submission, fmt.Errorf("error: unmarshalling ratings: %w", err)
		}
	}
	if keywords != nil {
		err = json.Unmarshal(keywords, &submission.Keywords)
		if err != nil {
			return submission, fmt.Errorf("error: unmarshalling keywords: %w", err)
		}
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
	var dateClosed *string
	var labels []byte
	var flags []byte
	var responses []byte
	var submissionIDs []byte
	var involved []byte

	err := db.QueryRowContext(db.context, selectTicketByID, ticketID).Scan(
		&ticket.ID, &ticket.Subject, &dateOpened, &dateClosed,
		&ticket.Status, &labels, &ticket.Priority, &flags, &ticket.Closed,
		&responses, &submissionIDs, &ticket.AssignedID, &involved,
	)
	if err != nil {
		return ticket, err
	}

	err = Scan(map[any]any{
		&ticket.DateOpened:    dateOpened,
		&ticket.DateClosed:    dateClosed,
		&ticket.Labels:        labels,
		&ticket.Flags:         flags,
		&ticket.Responses:     responses,
		&ticket.SubmissionIDs: submissionIDs,
		&ticket.UsersInvolved: involved,
	})

	return ticket, nil
}

// GetTicketsByAuditor returns a submissions of tickets by auditor id
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
func (db Sqlite) GetAllTickets() ([]Ticket, error) {
	return db.ticketsByQuery(selectAllTickets)
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
		var dateClosed *string
		var labels []byte
		var flags []byte
		var responses []byte
		var submissionIDs []byte
		var involved []byte

		err := rows.Scan(
			&ticket.ID, &ticket.Subject, &dateOpened, &dateClosed,
			&ticket.Status, &labels, &ticket.Priority, &flags, &ticket.Closed,
			&responses, &submissionIDs, &ticket.AssignedID, &involved,
		)
		if err != nil {
			return nil, err
		}

		err = Scan(map[any]any{
			&ticket.DateOpened:    dateOpened,
			&ticket.DateClosed:    dateClosed,
			&ticket.Labels:        labels,
			&ticket.Flags:         flags,
			&ticket.Responses:     responses,
			&ticket.SubmissionIDs: submissionIDs,
			&ticket.UsersInvolved: involved,
		})
		if err != nil {
			return nil, fmt.Errorf("error: scanning ticket rows: %w", err)
		}

		tickets = append(tickets, ticket)
	}

	if len(tickets) == 0 {
		return nil, fmt.Errorf("finished querying tickets by auditor: %w", sql.ErrNoRows)
	}

	return tickets, nil
}

// Scan scans the map of addresses to values and sets the values to the addresses.
// It expects the values to be []byte for JSON unmarshalling, or string for time.Time
func Scan(scan map[any]any) error {
	for key, value := range scan {
		k := reflect.ValueOf(key)
		if k.Kind() != reflect.Pointer {
			return fmt.Errorf("error: key %T is not a pointer", key)
		}

		if k.IsNil() {
			return fmt.Errorf("error: key %T is nil", key)
		}

		p := k.Elem()
		if value == nil {
			p.Set(reflect.Zero(p.Type()))
			continue
		}

		v := reflect.ValueOf(value)
		if p.Kind() == reflect.Pointer {
			if v.Kind() == reflect.Pointer && v.IsNil() {
				p.Set(reflect.Zero(p.Type()))
				continue
			}
			// Allocate new memory if nil
			if p.IsNil() {
				p.Set(reflect.New(p.Type().Elem()))
			}
			p = p.Elem() // Dereference once to get to the actual target
		}

		if data, ok := value.([]byte); ok {
			if len(data) == 0 || bytes.Equal(data, []byte("null")) {
				p.Set(reflect.Zero(p.Type()))
				continue
			}
			if key, ok := key.(*[]byte); ok {
				*key = data
				continue
			} else {
				if err := json.Unmarshal(data, p.Addr().Interface()); err != nil {
					return fmt.Errorf("error unmarshalling JSON: %w", err)
				}
				continue
			}
		}

		if v.Kind() == reflect.Pointer && p.Kind() != reflect.Pointer {
			// If the value is a pointer but the element is not, dereference the value first
			v = v.Elem()
		} else if p.Kind() == reflect.Pointer && v.Kind() != reflect.Pointer {
			// If the element is a pointer but the value is not, allocate a new pointer
			newVal := reflect.New(v.Type())
			newVal.Elem().Set(v)
			v = newVal
		}

		if !v.IsValid() {
			p.Set(reflect.Zero(p.Type()))
			continue
		}

		// Direct assignment if types are compatible
		if v.Type().AssignableTo(p.Type()) {
			p.Set(v)
			continue
		}

		// If the value is a string or *string and the element is a time.Time, parse the time
		if p.Type().AssignableTo(reflect.TypeFor[time.Time]()) {
			if v.Kind() == reflect.String {
				t, err := time.Parse(time.RFC3339Nano, v.String())
				if err != nil {
					return fmt.Errorf("error parsing time: %w", err)
				}
				p.Set(reflect.ValueOf(t))
				continue
			}
		}

		// If none of the above cases apply, it's an unsupported type combination
		return fmt.Errorf("error: unsupported type combination for key %T and value %T", key, value)
	}

	return nil
}

func (db Sqlite) GetAuditorFromHash(hash string) (Auditor, error) {
	var id int64
	err := db.QueryRowContext(db.context, SelectAuditorIDFromHash, hash).Scan(&id)
	if err != nil {
		return Auditor{}, err
	}

	return db.GetAuditorByID(id)
}

func (db Sqlite) GetUserIDFromSID(sid string) (int64, error) {
	var id int64
	err := db.QueryRowContext(db.context, SelectAuditorIDFromHash, Hash(sid)).Scan(&id)
	return id, err
}

func (db Sqlite) GetHashesFromID(userID int64) (HashID, error) {
	var hashes HashID

	rows, err := db.QueryContext(db.context, selectSIDsFromAuditorID, userID)
	if err != nil {
		return hashes, err
	}
	defer rows.Close()

	for rows.Next() {
		var sid SIDHash
		if err := rows.Scan(&sid.Hash, &sid.AuditorID); err != nil {
			return hashes, err
		}
		if hashes == nil {
			hashes = make(HashID)
		}
		hashes[sid.Hash] = sid.AuditorID
	}

	return hashes, nil
}

func (db Sqlite) ValidSID(user api.Credentials) bool {
	// warning: query row for some reason bugs out, either SQLITE_BUSY or sids table does not exist
	//row := db.QueryRow(selectSIDsFromHash, hash(user.Sid))
	//return row.Err() == nil
	// use Query instead

	if user.Sid == "" {
		return false
	}
	rows, err := db.QueryContext(db.context, selectSIDsFromHash, Hash(user.Sid))
	if err != nil {
		return false
	}
	defer rows.Close()

	return rows.Next()
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

// GetKnownModels returns a map of model hashes to a submissions of known model names
func (db Sqlite) GetKnownModels() (ModelHashes, error) {
	var modelHashes ModelHashes

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
		var bin []byte
		err = rows.Scan(&hash, &bin)
		if err != nil {
			return modelHashes, err
		}
		var models []string
		err = json.Unmarshal(bin, &models)
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

func (db Sqlite) AllAuditors() []Auditor {
	rows, err := db.QueryContext(db.context, selectAllAuditors)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var auditors []Auditor
	for rows.Next() {
		var auditor Auditor
		var role string
		err := rows.Scan(&auditor.UserID, &auditor.Username, &role, &auditor.AuditCount)
		if err != nil {
			return nil
		}
		auditor.Role = RoleLevel(role)
		auditors = append(auditors, auditor)
	}

	return auditors
}

func (db Sqlite) AllArtists() []Artist {
	rows, err := db.QueryContext(db.context, selectArtists)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var artists []Artist
	for rows.Next() {
		var artist Artist
		err := rows.Scan(&artist.Username, &artist.UserID)
		if err != nil {
			return nil
		}
		artists = append(artists, artist)
	}

	return artists
}
