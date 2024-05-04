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
	"regexp"
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

	// editAuditorRole statement for Auditor
	editAuditorRole = `UPDATE auditors SET role = ? WHERE auditor_id = ?;`

	// deleteAuditor statement for Auditor
	deleteAuditor = `DELETE FROM auditors WHERE auditor_id = ?;`

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
							 metadata, ratings, keywords, files)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(submission_id)
		DO UPDATE SET
					  user_id=excluded.user_id,
					  url=excluded.url,
					  audit_id=excluded.audit_id,
					  title=excluded.title,
					  description=excluded.description,
					  updated_at=excluded.updated_at,
					  metadata=excluded.metadata,
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
	INSERT INTO tickets (ticket_id, subject, date_opened, date_closed, status, labels, priority, flags, closed, responses, submissions_ids, auditor_id, involved)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(ticket_id)
		DO UPDATE SET
					  subject=excluded.subject,
					  date_opened=excluded.date_opened,
					  date_closed=excluded.date_closed,
					  status=excluded.status,
					  labels=excluded.labels,
					  priority=excluded.priority,
					  flags=excluded.flags,
					  closed=excluded.closed,
					  responses=excluded.responses,
					  submissions_ids=excluded.submissions_ids,
					  auditor_id=excluded.auditor_id,
					  involved=excluded.involved;
	`

	// newTicket statement for Ticket
	newTicket = `
	INSERT INTO tickets (subject, date_opened, date_closed, status, labels, priority, flags, closed, responses, submissions_ids, auditor_id, involved)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

	// deleteTicket statement for Ticket
	deleteTicket = `DELETE FROM tickets WHERE ticket_id = ?;`

	// insertSIDHash statement for SIDHash
	insertSIDHash = `
	INSERT INTO sids (sid_hash, auditor_id) VALUES (?, ?)
	ON CONFLICT(sid_hash) DO UPDATE SET sid_hash=excluded.sid_hash, auditor_id=excluded.auditor_id;
	`

	// deleteSIDHash statement for SIDHash
	deleteSIDHash = `
	DELETE FROM sids WHERE sid_hash = ?;
	`

	// deleteSIDHashes statement for SIDHash
	deleteSIDHashes = `
	DELETE FROM sids WHERE auditor_id = ?;
	`

	// upsertModel statement for ModelHashes
	upsertModel = `
	INSERT INTO models (hash, models) VALUES (?, ?)
	ON CONFLICT(hash) DO UPDATE SET models=excluded.models;
	`

	// upsertArtist statement for Artist
	upsertArtist = `
	INSERT INTO artists (username, user_id) VALUES (?, ?)
	ON CONFLICT(username) DO UPDATE SET user_id=excluded.user_id;
	`

	// deleteArtist statement for Artist
	deleteArtist         = `DELETE FROM artists WHERE user_id = ?;`
	deleteArtistUsername = `DELETE FROM artists WHERE username = ?;`
)

func (db Sqlite) InsertAuditor(auditor Auditor) error {
	_, err := db.ExecContext(db.context, upsertAuditor,
		auditor.UserID, auditor.Username, auditor.Role.String(), auditor.AuditCount,
	)

	return err
}

func (db Sqlite) EditAuditorRole(auditorID int64, role Role) error {
	_, err := db.ExecContext(db.context, editAuditorRole, role.String(), auditorID)
	return err
}

func (db Sqlite) DeleteAuditor(auditorID int64) error {
	_, err := db.ExecContext(db.context, deleteAuditor, auditorID)
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

// InkbunnyTimeLayout e.g. 2010-03-03 13:26:46.357649+00
const InkbunnyTimeLayout = "2006-01-02 15:04:05.999999-07"

func InkbunnySubmissionToDBSubmission(submission api.Submission) Submission {
	id, _ := strconv.ParseInt(submission.SubmissionID, 10, 64)
	userID, _ := strconv.ParseInt(submission.UserID, 10, 64)

	parsedTime, err := time.Parse(InkbunnyTimeLayout, submission.UpdateDateSystem)
	if err != nil {
		log.Printf("error: parsing date: %v", err)
		parsedTime = time.Now().UTC()
	}

	dbSubmission := Submission{
		ID:          id,
		UserID:      userID,
		Username:    submission.Username,
		URL:         fmt.Sprintf("https://inkbunny.net/s/%v", id),
		Title:       submission.Title,
		Description: submission.Description,
		Updated:     parsedTime,
		Ratings:     submission.Ratings,
		Keywords:    submission.Keywords,
	}

	for _, f := range submission.Files {
		dbSubmission.Files = append(dbSubmission.Files, File{
			File:    f,
			Caption: nil,
		})
	}

	SetSubmissionMeta(&dbSubmission)

	return dbSubmission
}

// The keyword IDs from Inkbunny
const (
	AIID              = "10503" // Deprecated: too generic, use AIGeneratedID
	AIGeneratedID     = "530560"
	AIAssistedID      = "677476"
	ComfyUIID         = "767686"
	ComfyUI           = "704819"
	Img2ImgID         = "730314"
	StableDiffusionID = "672195"
	AIArt             = "672082"
)

var PrivateTools = regexp.MustCompile(`\b(midjourney|novelai|bing|dall[- ]?e|nijijourney|craiyon)\b`)

// SetSubmissionMeta modifies a submission's Metadata based on its Keywords and other fields.
func SetSubmissionMeta(submission *Submission) {
	if submission == nil {
		return
	}
	for _, keyword := range submission.Keywords {
		switch keyword.KeywordName {
		case "ai generated", "ai art":
			submission.Metadata.Generated = true
			submission.Metadata.AIKeywords = append(submission.Metadata.AIKeywords, keyword.KeywordName)
			submission.Metadata.AISubmission = true
		case "ai assisted":
			submission.Metadata.Assisted = true
			submission.Metadata.AIKeywords = append(submission.Metadata.AIKeywords, keyword.KeywordName)
			submission.Metadata.AISubmission = true
		case "img2img":
			submission.Metadata.Img2Img = true
			submission.Metadata.AIKeywords = append(submission.Metadata.AIKeywords, keyword.KeywordName)
			submission.Metadata.AISubmission = true
		case "stable diffusion":
			submission.Metadata.StableDiffusion = true
			submission.Metadata.AIKeywords = append(submission.Metadata.AIKeywords, keyword.KeywordName)
			submission.Metadata.AISubmission = true
		case "comfyui", "comfy ui":
			submission.Metadata.ComfyUI = true
			submission.Metadata.AIKeywords = append(submission.Metadata.AIKeywords, keyword.KeywordName)
			submission.Metadata.AISubmission = true
		case "human":
			submission.Metadata.TaggedHuman = true
		}
		switch keyword.KeywordID {
		case AIGeneratedID, AIArt:
			submission.Metadata.Generated = true
		case AIAssistedID:
			submission.Metadata.Assisted = true
		case Img2ImgID:
			submission.Metadata.Img2Img = true
		case StableDiffusionID:
			submission.Metadata.StableDiffusion = true
		case ComfyUIID, ComfyUI:
			submission.Metadata.ComfyUI = true
		}
	}

	if tool := PrivateTools.FindString(submission.Description); tool != "" {
		submission.Metadata.AISubmission = true
		submission.Metadata.PrivateTool = true
		submission.Metadata.Generator = tool
	}

	var images int
	for _, file := range submission.Files {
		if strings.HasPrefix(file.File.MimeType, "image") {
			images++
			continue
		}
		if file.File.MimeType == "text/plain" {
			submission.Metadata.HasTxt = true
			continue
		}
		if file.File.MimeType == "application/json" {
			submission.Metadata.HasJSON = true
			continue
		}
	}
	if images > 1 {
		submission.Metadata.MultipleImages = true
	}
	submission.Metadata.MissingPrompt = true
	submission.Metadata.MissingModel = true
	if submission.Metadata.Objects != nil {
		submission.Metadata.AISubmission = true
		for _, obj := range submission.Metadata.Objects {
			if obj.Prompt != "" {
				submission.Metadata.MissingPrompt = false
			}
			if obj.OverrideSettings.SDModelCheckpoint != nil || obj.OverrideSettings.SDCheckpointHash != "" {
				submission.Metadata.MissingModel = false
			}
		}
	}
	if submission.Metadata.Params != nil && len(*submission.Metadata.Params) > 0 {
		submission.Metadata.AISubmission = true
	}
	if aiRegex.MatchString(submission.Title) {
		submission.Metadata.AITitle = true
		submission.Metadata.AISubmission = true
	}
	if strings.Contains(submission.Description, "stable diffusion") {
		submission.Metadata.StableDiffusion = true
		submission.Metadata.AIDescription = true
		submission.Metadata.AISubmission = true
	}
	if strings.Contains(submission.Description, "comfyui") {
		submission.Metadata.ComfyUI = true
		submission.Metadata.AIDescription = true
		submission.Metadata.AISubmission = true
	}
	if submission.Metadata.AISubmission && len(submission.Metadata.AIKeywords) == 0 {
		submission.Metadata.MissingTags = true
	}
}

var aiRegex = regexp.MustCompile(`(?i)\b(ai|ia|ai generated|ai assisted|img2img|stable diffusion|comfyui)\b`)

var payment = regexp.MustCompile(`(?i)\b(ko-?fi|paypal|patreon|subscribestar|donate|bitcoin|ethereum|monero)\b`)

func TicketLabels(submission Submission) []TicketLabel {
	labels := make(map[TicketLabel]bool)
	metadata := submission.Metadata

	if metadata.TaggedHuman {
		labels[LabelTaggedHuman] = true
	}
	if metadata.DetectedHuman {
		labels[LabelDetectedHuman] = true
	}

	if metadata.AISubmission {
		if len(metadata.Objects) == 0 {
			if metadata.HasTxt || metadata.HasJSON {
				labels[LabelCannotParse] = true
			} else {
				labels[LabelMissingParams] = true
			}
		}
		if metadata.MissingTags {
			labels[LabelMissingTags] = true
		}
		if len(metadata.ArtistUsed) > 0 {
			labels[LabelArtistUsed] = true
		}

		if payment.MatchString(submission.Description) {
			labels[LabelPayMention] = true
		}

		if submission.Updated.Before(Nov21) {
			labels[LabelBeforeRuleRevision] = true
		}

		if metadata.PrivateTool {
			labels[LabelPrivateTool] = true
		}

		const (
			prompt = "prompt"
			model  = "model"
			seed   = "seed"
		)
		hint := [3]struct {
			label   string
			missing bool
			partial bool
		}{
			{label: prompt},
			{label: model},
			{label: seed},
		}
		for _, obj := range metadata.Objects {
			if obj.Prompt == "" {
				hint[0].missing = true
			} else {
				hint[0].partial = true
			}
			if obj.OverrideSettings.SDModelCheckpoint == nil && obj.OverrideSettings.SDCheckpointHash == "" {
				hint[1].missing = true
			} else {
				hint[1].partial = true
			}
			if obj.Seed == 0 {
				hint[2].missing = true
			} else {
				hint[2].partial = true
			}
		}
		for _, v := range hint {
			if v.missing {
				if v.partial {
					labels[TicketLabel("partial_"+v.label)] = true
				} else {
					labels[TicketLabel("missing_"+v.label)] = true
				}
			}
		}
	}
	out := make([]TicketLabel, 0, len(labels))
	for label := range labels {
		out = append(out, label)
	}
	return out
}

func (db Sqlite) InsertSubmission(submission Submission) error {
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

	args, err := assertArgs(
		submission.ID, submission.UserID, submission.URL, submission.AuditID,
		submission.Title, submission.Description, submission.Updated,
		submission.Metadata, submission.Ratings, submission.Keywords, submission.Files,
	)
	if err != nil {
		return fmt.Errorf("error: asserting metadata: %w", err)
	}

	_, err = db.ExecContext(db.context, upsertSubmission, args...)
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
func (db Sqlite) InsertTicket(ticket Ticket, force ...bool) (int64, error) {
	if len(force) > 0 && force[0] {
		ticket.ID = 0
	}
	if ticket.ID != 0 {
		return 0, ErrTicketIsSet
	}
	return db.UpsertTicket(ticket)
}

// UpsertTicket inserts or updates a ticket in the database.
// If the ticket ID is unset, it will insert a new ticket.
func (db Sqlite) UpsertTicket(ticket Ticket) (int64, error) {
	args, err := assertArgs(
		ticket.ID, ticket.Subject,
		ticket.DateOpened, ticket.DateClosed,
		ticket.Status, ticket.Labels, ticket.Priority, ticket.Flags, ticket.Closed,
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
		return 0, fmt.Errorf("error: %vserting ticket: %w", process, err)
	}

	if isInsert {
		id, err := res.LastInsertId()
		if err != nil && id != ticket.ID {
			return 0, fmt.Errorf("error: last insert id %v does not match ticket id: %v: %w", id, ticket.ID, err)
		}
	}

	return ticket.ID, nil
}

func (db Sqlite) DeleteTicket(id int64) error {
	result, err := db.ExecContext(db.context, deleteTicket, id)
	if err != nil {
		return fmt.Errorf("error: deleting ticket: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error: getting rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("error: ticket %d doesn't exist in the database", id)
	}

	return nil
}

// assertArgs asserts that the arguments are valid sqlite types and marshals them if necessary.
// TODO: Include creation query to check what type is expected.
func assertArgs(args ...any) ([]any, error) {
	for i := range args {
		if args[i] == nil {
			continue
		}
		var length = -1
		switch a := args[i].(type) {
		case *string, *int, *int64, *float32, *float64, *bool:
			if a == nil {
				args[i] = nil
			}
		case string, int, int64, float32, float64, bool:

		case *[]byte, []byte:
			if a == nil {
				args[i] = nil
			}
		case nil:
			args[i] = nil
		case time.Time:
			args[i] = parseTime(a)
		case *time.Time:
			if a == nil {
				args[i] = nil
				continue
			}
			args[i] = parseTime(a)
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

			if isNil(v) {
				args[i] = nil
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

func parseTime(t interface{ UTC() time.Time }) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func isNil(v reflect.Value) bool {
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return true
		}
		return isNil(v.Elem())
	}
	if slices.Contains([]reflect.Kind{
		reflect.Array, reflect.Slice, reflect.Map,
		reflect.Chan, reflect.Func,
	}, v.Kind()) {
		return v.IsNil()
	}
	return false
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
	_, err := db.ExecContext(db.context, insertSIDHash, sid.Hash, sid.AuditorID)
	return err
}

func (db Sqlite) RemoveSIDHash(sid SIDHash) error {
	if sid.Hash == Hash("") {
		return fmt.Errorf("error: sid hash cannot be empty")
	}
	res, err := db.ExecContext(db.context, deleteSIDHash, sid.Hash)
	if err != nil {
		return err
	}
	r, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if r == 0 {
		return fmt.Errorf("error: no rows affected")
	}

	return nil
}

func (db Sqlite) LogoutAll(sid SIDHash) error {
	if sid.Hash == Hash("") {
		return fmt.Errorf("error: sid hash cannot be empty")
	}
	id, err := db.GetUserIDFromSID(sid.Hash)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(db.context, deleteSIDHashes, id)
	return err
}

func HashCredentials(user api.Credentials) SIDHash {
	return SIDHash{
		Hash:      Hash(user.Sid),
		AuditorID: int64(user.UserID.Int()),
	}
}

func Hash(s any) hashedSID {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", s)))
	return hashedSID(fmt.Sprintf("%x", h.Sum(nil)))
}

type hashedSID = string

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

func (db Sqlite) UpsertModel(models ModelHashes) error {
	if models == nil {
		return nil
	}

	for hash, model := range models {
		stored := db.ModelNamesFromHash(hash)
		var appended bool
		for _, newModel := range model {
			if newModel == "" {
				continue
			}
			if slices.Contains(stored, newModel) {
				continue
			}
			stored = append(stored, newModel)
			appended = true
		}
		if !appended {
			continue
		}

		marshal, err := json.Marshal(stored)
		if err != nil {
			return fmt.Errorf("error: marshalling model: %w", err)
		}
		_, err = db.ExecContext(db.context, upsertModel, hash, marshal)
		if err != nil {
			return fmt.Errorf("error: upserting model: %w", err)
		}
	}

	return nil
}

func (db Sqlite) UpsertArtist(artists ...Artist) error {
	if len(artists) == 0 {
		return nil
	}

	for _, artist := range artists {
		_, err := db.ExecContext(db.context, upsertArtist, artist.Username, artist.UserID)
		if err != nil {
			return fmt.Errorf("error: upserting artist: %w", err)
		}
	}

	return nil
}

func (db Sqlite) DeleteArtist(username string) error {
	_, err := db.ExecContext(db.context, deleteArtistUsername, username)
	return err
}
