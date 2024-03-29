package db

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ellypaws/inkbunny/api"
	"log"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Insert statements
const (
	// upsertAuditor statement for Auditor
	upsertAuditor = `
	INSERT INTO auditors (auditor_id, username, role, audit_count) VALUES (?, ?, ?, ?)
	ON CONFLICT(auditor_id) DO UPDATE SET username=excluded.username, role=excluded.role, audit_count=excluded.audit_count;
	`

	// increaseAuditCount statement for Auditor
	increaseAuditCount = `
	UPDATE auditors SET audit_count = audit_count + 1 WHERE auditor_id = ?;
	`

	// updateAuditCount statement for Auditor
	updateAuditCount = `
	UPDATE auditors SET audit_count = ? WHERE auditor_id = ?;
	`

	// upsertAudit statement for Audit
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

	// updateAuditID statement for Audit
	updateAuditID = `
	UPDATE audits SET audit_id = ? WHERE submission_id = ?;
	`

	// updateSubmissionFile statement for Submission
	updateSubmissionFile = `
	UPDATE submissions SET files = ? WHERE submission_id = ?;
	`

	// upsertSubmission statement for Submission
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

	// updateSubmissionDescription statement for Submission
	updateSubmissionDescription = `
	UPDATE submissions SET description = ? WHERE submission_id = ?;
	`

	// updateSubmissionAudit statement for Submission
	updateSubmissionAudit = `
	UPDATE submissions SET audit_id = ? WHERE submission_id = ?;
	`

	// upsertTicket statement for Ticket
	upsertTicket = `
	INSERT INTO tickets (ticket_id, subject, date_opened, status, labels, priority, closed, responses, submissions_ids, auditor_id, involved)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(ticket_id)
		DO UPDATE SET
					  subject=excluded.subject,
					  date_opened=excluded.date_opened,
					  status=excluded.status,
					  labels=excluded.labels,
					  priority=excluded.priority,
					  closed=excluded.closed,
					  responses=excluded.responses,
					  submissions_ids=excluded.submissions_ids,
					  auditor_id=excluded.auditor_id,
					  involved=excluded.involved;
	`

	// newTicket statement for Ticket
	newTicket = `
	INSERT INTO tickets (subject, date_opened, status, labels, priority, closed, responses, submissions_ids, auditor_id, involved)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

	// insertSIDHash statement for SIDHash
	insertSIDHash = `
	INSERT INTO sids (user_id, username, sid_hash) VALUES (?, ?, ?)
	ON CONFLICT(user_id) DO UPDATE SET username=excluded.username, sid_hash=excluded.sid_hash;
	`

	// upsertModel statement for ModelHashes
	upsertModel = `
	INSERT INTO models (hash, models) VALUES (?, ?)
	ON CONFLICT(hash) DO UPDATE SET models=excluded.models;
	`
)

func (db Sqlite) InsertAuditor(auditor Auditor) error {
	_, err := db.ExecContext(db.context, upsertAuditor,
		auditor.UserID, auditor.Username, auditor.Role.String(), auditor.AuditCount,
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
// Audit.AuditorID needs to be non-nil and exist in the database before inserting an audit.
// Similarly, the Submission needs to be in the database as well and be filled in the audit.
// If successful, the submission will be updated with the new audit_id.
func (db Sqlite) InsertAudit(audit Audit) (int64, error) {
	if audit.AuditorID == nil {
		if audit.auditor.UserID == 0 {
			return 0, errors.New("error: auditor id cannot be nil")
		}
		// if auditor id is populated, use as fallback
		audit.AuditorID = &audit.auditor.UserID
	}

	var flags []string
	for _, flag := range audit.Flags {
		flags = append(flags, string(flag))
	}

	res, err := db.ExecContext(db.context, upsertAudit,
		audit.AuditorID,
		audit.SubmissionID, audit.SubmissionUsername, audit.SubmissionUserID,
		strings.Join(flags, ","), audit.ActionTaken,
	)
	if err != nil {
		return 0, fmt.Errorf("error: inserting audit: %w", err)
	}

	audit, err = db.GetAuditBySubmissionID(audit.SubmissionID)
	if err != nil {
		return 0, fmt.Errorf("error: getting audit by submission id: %w", err)
	}

	if id, err := res.LastInsertId(); err != nil && id != audit.ID() {
		return 0, fmt.Errorf("error: last insert id does not match audit id: %w", err)
	}

	// set audit in submission if it exists in the database
	res, err = db.ExecContext(db.context, updateSubmissionAudit, audit.id, audit.SubmissionID)
	if err != nil {
		return 0, fmt.Errorf("error: updating submission audit: %w", err)
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		log.Printf("error: getting rows affected: %v", err)
		return 0, err
	}

	if rowCount == 0 {
		log.Printf("warning: submission %d doesn't exist in the database", audit.SubmissionID)
	}

	return audit.id, nil
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
		var audit Audit
		err = rows.Scan(&audit.id, &audit.SubmissionID)
		if err != nil {
			return err
		}

		audits = append(audits, audit)
	}

	for _, audit := range audits {
		_, err = db.ExecContext(db.context, updateSubmissionAudit, audit.id, audit.SubmissionID)
		if err != nil {
			return fmt.Errorf("error: updating submission audit: %w", err)
		}
	}

	return nil
}

func (db Sqlite) InsertFile(file File) error {
	if file.File.SubmissionID == "" {
		return errors.New("error: submission id is empty")
	}

	marshal, err := json.Marshal(file)
	if err != nil {
		return fmt.Errorf("error: marshalling file: %w", err)
	}

	_, err = db.ExecContext(db.context, updateSubmissionFile,
		marshal, file.File.SubmissionID,
	)

	return err
}

// e.g. 2010-03-03 13:26:46.357649+00
const inkbunnyTimeLayout = "2006-01-02 15:04:05.999999-07"

func InkbunnySubmissionToDBSubmission(submission api.Submission) Submission {
	id, _ := strconv.ParseInt(submission.SubmissionID, 10, 64)
	userID, _ := strconv.ParseInt(submission.UserID, 10, 64)

	parsedTime, err := time.Parse(inkbunnyTimeLayout, submission.UpdateDateSystem)
	if err != nil {
		log.Printf("error: parsing date: %v", err)
		parsedTime = time.Now().UTC()
	}

	dbSubmission := Submission{
		ID:          id,
		UserID:      userID,
		URL:         fmt.Sprintf("https://inkbunny.net/s/%v", id),
		Title:       submission.Title,
		Description: submission.Description,
		Updated:     parsedTime,
		Ratings:     submission.Ratings,
		Keywords:    submission.Keywords,
	}

	SetTagsFromKeywords(&dbSubmission)

	for _, f := range submission.Files {
		dbSubmission.Files = append(dbSubmission.Files, File{
			File: f,
			Info: nil,
			Blob: nil,
		})
	}

	return dbSubmission
}

func SetTagsFromKeywords(submission *Submission) {
	if submission == nil {
		return
	}
	for _, keyword := range submission.Keywords {
		switch strings.ReplaceAll(keyword.KeywordName, " ", "_") {
		case "ai_generated":
			submission.Generated = true
		case "ai_assisted":
			submission.Assisted = true
		case "img2img":
			submission.Img2Img = true
		}
	}
}

func SubmissionLabels(submission Submission) []TicketLabel {
	var labels []TicketLabel
	if submission.Generated {
		labels = append(labels, LabelAIGenerated)
	}
	if submission.Assisted {
		labels = append(labels, LabelAIAssisted)
	}
	if submission.Img2Img {
		labels = append(labels, LabelImg2Img)
	}
	return labels
}

func (db Sqlite) InsertSubmission(submission Submission) error {
	ratings, err := json.Marshal(submission.Ratings)
	if err != nil {
		return fmt.Errorf("error: marshalling ratings: %w", err)
	}

	keywords, err := json.Marshal(submission.Keywords)
	if err != nil {
		return fmt.Errorf("error: marshalling keywords: %w", err)
	}

	var filesMarshal sql.RawBytes
	if len(submission.Files) > 0 {
		filesMarshal, err = json.Marshal(submission.Files)
		if err != nil {
			return fmt.Errorf("error: marshalling files: %w", err)
		}
	}

	if submission.AuditID == nil {
		audit, err := db.GetAuditBySubmissionID(submission.ID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("error: getting audit by submission id: %w", err)
			}
		} else {
			submission.AuditID = &audit.id
			submission.audit = &audit
		}
	}

	_, err = db.ExecContext(db.context, upsertSubmission,
		submission.ID, submission.UserID, submission.URL, submission.AuditID,
		submission.Title, submission.Description, submission.Updated.UTC().Format(time.RFC3339Nano),
		submission.Generated, submission.Assisted, submission.Img2Img,
		ratings, keywords, filesMarshal,
	)
	if err != nil {
		return fmt.Errorf("error: inserting submission: %w", err)
	}

	return nil
}

func (db Sqlite) UpdateDescription(submission Submission) error {
	_, err := db.ExecContext(db.context, updateSubmissionDescription, submission.Description, submission.ID)
	return err
}

// ErrTicketIsSet is returned when a ticket ID is set but Sqlite.InsertTicket was called.
// Use Sqlite.UpsertTicket instead.
var ErrTicketIsSet = errors.New("error: ticket id is set but InsertTicket was called")

// InsertTicket inserts a new ticket into the database.
// The ID is expected to be non-zero as it's a new ticket.
// This ensures that InsertTicket is only for new tickets.
// Set force to true to unset the ticket ID and always insert a new ticket.
func (db Sqlite) InsertTicket(ticket Ticket, force ...bool) error {
	if len(force) > 0 && force[0] {
		ticket.ID = 0
	}
	if ticket.ID != 0 {
		return ErrTicketIsSet
	}
	return db.UpsertTicket(ticket)
}

// UpsertTicket inserts or updates a ticket in the database.
// If the ticket ID is unset, it will insert a new ticket.
func (db Sqlite) UpsertTicket(ticket Ticket) error {
	args, err := assertArgs(
		ticket.ID, ticket.Subject, ticket.DateOpened.UTC().Format(time.RFC3339Nano),
		ticket.Status, ticket.Labels, ticket.Priority, ticket.Closed,
		ticket.Responses, ticket.SubmissionIDs, ticket.AssignedID, ticket.UsersInvolved,
	)

	var isInsert bool = ticket.ID == 0
	var query string = upsertTicket
	if isInsert {
		query = newTicket
		args = args[1:]
	}
	res, err := db.ExecContext(db.context, query, args...)
	if err != nil {
		var process string = "up"
		if isInsert {
			process = "in"
		}
		return fmt.Errorf("error: %vserting ticket: %w", process, err)
	}

	if id, err := res.LastInsertId(); err != nil && id != ticket.ID {
		return fmt.Errorf("error: last insert id does not match ticket id: %w", err)
	}

	return nil
}

// assertArgs asserts that the arguments are valid sqlite types and marshals them if necessary.
func assertArgs(args ...any) ([]any, error) {
	for i := range args {
		var length = -1
		switch a := args[i].(type) {
		case *string, *int, *int64, *float32, *float64, *bool:

		case string, int, int64, float32, float64, bool:

		case *[]byte, []byte:

		case nil:
			args[i] = nil
		default:
			// use reflect to check if it's a slice
			v := reflect.ValueOf(a)

			if !slices.Contains([]reflect.Kind{
				reflect.Array,
				reflect.Func,
				reflect.Map,
				reflect.Pointer,
				reflect.Slice,
				reflect.Struct,
				reflect.UnsafePointer,
			}, v.Kind()) {
				return nil, fmt.Errorf("error: invalid type: %T", a)
			}

			if slices.Contains([]reflect.Kind{
				reflect.Array, reflect.Slice, reflect.Map,
				reflect.Chan, reflect.Func, reflect.Interface, reflect.Pointer,
			}, v.Kind()) && v.IsNil() {
				continue
			}

			if slices.Contains([]reflect.Kind{
				reflect.Slice, reflect.Array, reflect.Map,
			}, v.Kind()) {
				length = v.Len()
			}

			var err error
			if length != -1 {
				args[i], err = marshal(args[i], length)
				if err != nil {
					return nil, fmt.Errorf("error: marshalling %#v: %w", args[i], err)
				}
				if b, ok := args[i].([]byte); !ok || bytes.Equal(b, []byte("null")) {
					args[i] = nil
				}
			} else {
				args[i], err = json.Marshal(a)
				if err != nil {
					return nil, fmt.Errorf("error: marshalling %#v: %w", a, err)
				}
			}
		}
	}
	return args, nil
}

func marshal(value any, length int) ([]byte, error) {
	var marshal []byte
	if value != nil && length > 0 {
		var err error
		marshal, err = json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("error: marshalling labels: %w", err)
		}
	}
	return marshal, nil
}

func (db Sqlite) InsertSIDHash(sid SIDHash) error {
	stored, err := db.GetSIDsFromUserID(sid.UserID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	var hashes hashmap = make(hashmap)
	if len(stored.hashes) > 0 {
		for hash := range stored.hashes {
			hashes[hash] = struct{}{}
		}
	}

	for hash := range sid.hashes {
		hashes[hash] = struct{}{}
	}

	var marshal []byte
	if len(hashes) > 0 {
		marshal, err = json.Marshal(hashes)
		if err != nil {
			return fmt.Errorf("error: marshalling hashes: %w", err)
		}
	}

	_, err = db.ExecContext(db.context, insertSIDHash, sid.UserID, sid.Username, marshal)
	return err
}

func (db Sqlite) RemoveSIDHash(sid SIDHash) error {
	stored, err := db.GetSIDsFromUserID(sid.UserID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil
	case err != nil:
		return fmt.Errorf("error: could not get hashes: %w", err)
	}

	for hashToRemove := range sid.hashes {
		delete(stored.hashes, hashToRemove)
	}

	return db.InsertSIDHash(stored)
}

func HashCredentials(user api.Credentials) SIDHash {
	checksum := hash(user.Sid)
	return SIDHash{
		UserID:   int64(user.UserID.Int()),
		Username: user.Username,
		hashes:   checksum,
	}
}

func (sidHash SIDHash) SetHash(sid string) SIDHash {
	checksum := hash(sid)
	sidHash.hashes = checksum
	return sidHash
}

func hash(s any) hashmap {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", s)))
	return hashmap{fmt.Sprintf("%x", h.Sum(nil)): struct{}{}}
}

func (db Sqlite) ValidSID(user api.Credentials) bool {
	stored, err := db.GetSIDsFromUserID(int64(user.UserID.Int()))
	if err != nil {
		return false
	}

	checksum := hash(user.Sid)
	for hash := range checksum {
		if _, ok := stored.hashes[hash]; ok {
			return true
		}
	}

	return false
}

func (db Sqlite) InsertModel(models ModelHashes) error {
	if models == nil {
		return nil
	}

	for hash, model := range models {
		marshal, err := json.Marshal(model)
		if err != nil {
			return fmt.Errorf("error: marshalling model: %w", err)
		}
		_, err = db.ExecContext(db.context, upsertModel, hash, marshal)
		if err != nil {
			return fmt.Errorf("error: inserting model: %w", err)
		}
	}

	return nil
}
