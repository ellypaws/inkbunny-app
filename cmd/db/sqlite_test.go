package db

import (
	"context"
	"database/sql"
	"github.com/ellypaws/inkbunny/api"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"
)

var db, _ = tempDB(context.Background())
var useVirtualDB = true
var slow = os.Getenv("SLOW") == "true"

const tempPhysicalDB = "temp.sqlite"

func tempDB(ctx context.Context) (*Sqlite, error) {
	if !useVirtualDB {
		return New(context.WithValue(ctx, "filename", tempPhysicalDB))
	}

	return New(context.WithValue(ctx, ":memory:", useVirtualDB))
}

func resetDB(t *testing.T) {
	var err error
	db, err = tempDB(context.Background())
	if err != nil {
		t.Fatalf("tempDB() failed: %v", err)
	}
}

func TestPhysical(t *testing.T) {
	db, err := New(context.WithValue(context.Background(), "filename", tempPhysicalDB))
	if db == nil {
		t.Fatal("New() failed: db is nil")
	}

	err = db.PingContext(db.context)
	if err != nil {
		t.Errorf("PingContext() failed: %v", err)
	}

	err = db.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	t.Log("TestPhysical() passed")
}

func TestNew(t *testing.T) {
	db, err := tempDB(context.Background())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if db == nil {
		t.Fatal("New() failed: db is nil")
	}

	err = db.PingContext(db.context)
	if err != nil {
		t.Errorf("PingContext() failed: %v", err)
	}

	err = db.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	t.Log("TestNew() passed")
}

func TestSqlite_InsertAuditor(t *testing.T) {
	resetDB(t)

	auditor := Auditor{
		UserID:   196417,
		Username: "Elly",
		Role:     RoleAuditor,
	}

	err := db.InsertAuditor(auditor)
	if err != nil {
		t.Fatalf("InsertAuditor() failed: %v", err)
	}

	t.Log("TestSqlite_InsertAuditor() passed")
}

func TestSqlite_IncreaseAuditCount(t *testing.T) {
	resetDB(t)

	auditor := Auditor{
		UserID:   196417,
		Username: "Elly",
		Role:     RoleAuditor,
	}

	err := db.InsertAuditor(auditor)
	if err != nil {
		t.Fatalf("InsertAuditor() failed: %v", err)
	}

	err = db.IncreaseAuditCount(auditor.UserID)
	if err != nil {
		t.Fatalf("IncreaseAuditCount() failed: %v", err)
	}

	auditor, err = db.GetAuditorByID(auditor.UserID)
	if err != nil {
		t.Fatalf("GetAuditorByID() failed: %v", err)
	}

	if auditor.AuditCount != 1 {
		t.Fatalf("IncreaseAuditCount() failed: expected 1, got %v", auditor.AuditCount)
	}

	t.Log("TestSqlite_IncreaseAuditCount() passed")
}

func TestSqlite_SyncAuditCount(t *testing.T) {
	resetDB(t)

	auditor := Auditor{
		UserID:   196417,
		Username: "Elly",
		Role:     RoleAuditor,
	}

	err := db.InsertAuditor(auditor)
	if err != nil {
		t.Fatalf("InsertAuditor() failed: %v", err)
	}

	err = db.IncreaseAuditCount(auditor.UserID)
	if err != nil {
		t.Fatalf("IncreaseAuditCount() failed: %v", err)
	}

	auditor, err = db.GetAuditorByID(auditor.UserID)
	if err != nil {
		t.Fatalf("GetAuditorByID() failed: %v", err)
	}

	if auditor.AuditCount != 1 {
		t.Fatalf("IncreaseAuditCount() failed: expected 1, got %v", auditor.AuditCount)
	}

	err = db.SyncAuditCount(auditor.UserID)
	if err != nil {
		t.Fatalf("SyncAuditCount() failed: %v", err)
	}

	auditor, err = db.GetAuditorByID(auditor.UserID)
	if err != nil {
		t.Fatalf("GetAuditorByID() failed: %v", err)
	}

	if useVirtualDB {
		if auditor.AuditCount != 0 {
			t.Errorf("SyncAuditCount() failed: expected 0, got %v", auditor.AuditCount)
		}
	} else {
		if !(auditor.AuditCount > 0) {
			t.Errorf("SyncAuditCount() failed: expected > 0, got %v", auditor.AuditCount)
		}
	}

	submission := Submission{
		ID:          123,
		UserID:      456,
		URL:         "url",
		Title:       "title",
		Description: "description",
		AuditID:     nil,
		Files: []File{
			{
				File: api.File{
					FileID:   "123",
					FileName: "file",
				},
				Info: &GenerationInfo{
					Generator: "generator",
					Model:     "model",
				},
				Blob: nil,
			},
		},
	}

	err = db.InsertSubmission(submission)
	if err != nil {
		t.Fatalf("InsertSubmission() failed: %v", err)
	}

	audit := Audit{
		AuditorID:          &auditor.UserID,
		SubmissionID:       123,
		SubmissionUsername: "User",
		SubmissionUserID:   456,
		Flags:              []Flag{FlagUndisclosed},
		ActionTaken:        "none",
	}

	auditID, err := db.InsertAudit(audit)
	if err != nil {
		t.Fatalf("InsertAudit() failed: %v", err)
	}

	checkSubmission, _ := db.GetSubmissionByID(123)
	if checkSubmission.AuditID == nil {
		t.Fatalf("InsertAudit() failed: expected non-nil audit, got nil")
	}

	if *checkSubmission.AuditID != auditID {
		t.Fatalf("InsertAudit() failed: expected %v, got %v", auditID, *checkSubmission.AuditID)
	}

	err = db.SyncAuditCount(auditor.UserID)
	if err != nil {
		t.Fatalf("SyncAuditCount() failed: %v", err)
	}

	auditor, err = db.GetAuditorByID(auditor.UserID)
	if err != nil {
		t.Fatalf("GetAuditorByID() failed: %v", err)
	}

	if !(auditor.AuditCount > 0) {
		t.Fatalf("SyncAuditCount() failed: expected > 0, got %v", auditor.AuditCount)
	}

	t.Log("TestSqlite_SyncAuditCount() passed")
}

func TestSqlite_GetAuditsByAuditor(t *testing.T) {
	if useVirtualDB == true {
		t.Skip("skipping test in virtual db")
	}

	audits, err := db.GetAuditsByAuditor(196417)
	if err != nil {
		t.Fatalf("GetAuditsByAuditor() failed: %v", err)
	}

	if useVirtualDB {
		if len(audits) != 0 {
			t.Fatalf("GetAuditsByAuditor() failed: expected 0, got %v", len(audits))
		}
	} else {
		if !(len(audits) > 0) {
			t.Fatalf("GetAuditsByAuditor() failed: expected > 0, got %v", len(audits))
		}
	}

	t.Log("TestSqlite_GetAuditsByAuditor() passed")
}

func TestSqlite_InsertFile(t *testing.T) {
	resetDB(t)

	file := File{
		File: api.File{
			FileID:       "123",
			FileName:     "file",
			SubmissionID: "123",
		},
		Info: &GenerationInfo{
			Generator: "generator",
			Model:     "model",
		},
		Blob: nil,
	}

	err := db.InsertFile(file)
	if err != nil {
		t.Fatalf("InsertFile() failed: %v", err)
	}

	t.Log("TestSqlite_InsertFile() passed")
}

func TestSqlite_InsertSubmission(t *testing.T) {
	resetDB(t)

	submission := Submission{
		ID:          123,
		UserID:      456,
		URL:         "url",
		Title:       "title",
		Description: "description",
	}

	err := db.InsertSubmission(submission)
	if err != nil {
		t.Fatalf("InsertSubmission() failed: %v", err)
	}

	t.Log("TestSqlite_InsertSubmission() passed")
}

func TestSqlite_InsertAudit(t *testing.T) {
	TestSqlite_GetAuditorByID(t)

	auditor, err := db.GetAuditorByID(196417)
	if err != nil {
		t.Fatalf("GetAuditorByID() failed: %v", err)
	}

	err = db.InsertSubmission(Submission{ID: 123})
	if err != nil {
		t.Fatalf("InsertSubmission() failed: %v", err)
	}

	audit := Audit{
		AuditorID:          &auditor.UserID,
		SubmissionID:       123,
		SubmissionUsername: "User",
		SubmissionUserID:   456,
		Flags:              []Flag{FlagUndisclosed},
		ActionTaken:        "none",
	}

	_, err = db.InsertAudit(audit)
	if err != nil {
		t.Fatalf("InsertAudit() failed: %v", err)
	}

	t.Log("TestSqlite_InsertAudit() passed")
}

func TestSqlite_GetAuditorByID(t *testing.T) {
	resetDB(t)

	auditor := Auditor{
		UserID:   196417,
		Username: "Elly",
		Role:     RoleAuditor,
	}

	err := db.InsertAuditor(auditor)
	if err != nil {
		t.Fatalf("InsertAuditor() failed: %v", err)
	}

	auditor, err = db.GetAuditorByID(auditor.UserID)
	if err != nil {
		t.Fatalf("GetAuditorByID() failed: %v", err)
	}
}

func TestSqlite_GetSubmissionByID(t *testing.T) {
	resetDB(t)

	submission := Submission{
		ID:          123,
		UserID:      456,
		URL:         "url",
		Title:       "title",
		Description: "description",
		Updated:     time.Now(),
	}

	err := db.InsertSubmission(submission)
	if err != nil {
		t.Fatalf("InsertSubmission() failed: %v", err)
	}

	submission, err = db.GetSubmissionByID(submission.ID)
	if err != nil {
		t.Fatalf("GetSubmissionByID() failed: %v", err)
	}

	t.Logf("TestSqlite_GetSubmissionByID() passed: %v", submission)
}

func TestSqlite_GetAuditBySubmissionID(t *testing.T) {
	resetDB(t)

	auditor := &Auditor{
		UserID:   196417,
		Username: "Elly",
		Role:     RoleAuditor,
	}

	err := db.InsertAuditor(*auditor)
	if err != nil {
		t.Fatalf("InsertAuditor() failed: %v", err)
	}

	submission := Submission{
		ID:          123,
		UserID:      456,
		URL:         "url",
		Title:       "title",
		Description: "description",
		Updated:     time.Now(),
	}

	err = db.InsertSubmission(submission)
	if err != nil {
		t.Fatalf("InsertSubmission() failed: %v", err)
	}

	audit := Audit{
		AuditorID:          &auditor.UserID,
		SubmissionID:       submission.ID,
		SubmissionUsername: "User",
		SubmissionUserID:   456,
		Flags:              []Flag{FlagUndisclosed},
		ActionTaken:        "none",
	}

	id, err := db.InsertAudit(audit)

	audit, err = db.GetAuditBySubmissionID(submission.ID)
	if err != nil {
		t.Fatalf("GetAuditBySubmissionID() failed: %v", err)
	}

	if audit.SubmissionID != submission.ID {
		t.Fatalf("GetAuditBySubmissionID() failed: expected %v, got %v", submission.ID, audit.SubmissionID)
	}

	submission, err = db.GetSubmissionByID(submission.ID)
	if err != nil {
		t.Fatalf("GetSubmissionByID() failed: %v", err)
	}

	if submission.audit == nil {
		t.Fatalf("GetAuditBySubmissionID() failed: expected non-nil audit, got nil")
	}

	if submission.AuditID == nil || *submission.AuditID != id {
		t.Fatalf("GetAuditBySubmissionID() failed: expected %v, got %v", audit.id, submission.AuditID)
	}
}

const descriptionSQLInjection = `description'); DROP TABLE submissions; --`
const userIDSQLInjection = `123 OR 1=1`

func TestSqlite_InsertSubmission_SQLInjection(t *testing.T) {
	resetDB(t)

	submission := Submission{
		ID:          123,
		UserID:      456,
		URL:         "url",
		Title:       "title",
		Description: descriptionSQLInjection,
	}

	err := db.InsertSubmission(submission)
	if err != nil {
		t.Fatalf("InsertSubmission() failed: %v", err)
	}

	submissions, err := db.GetSubmissionByID(submission.ID)
	if err != nil {
		t.Fatalf("GetSubmissionByID() failed: %v", err)
	}

	if submissions.Description != descriptionSQLInjection {
		t.Fatalf("InsertSubmission() failed: expected %v, got %v", descriptionSQLInjection, submissions.Description)
	}

	//	No longer possible as we're using int64 for submission_id instead of string
	//	submissions, err = db.GetSubmissionByID(userIDSQLInjection)
	//	if !errors.Is(err, sql.ErrNoRows) {
	//		t.Fatalf("GetSubmissions() failed: expected sql.ErrNoRows, got %v", err)
	//	}

	t.Logf("TestSqlite_InsertSubmission_SQLInjection() passed: %#v", submissions)
}

func TestSqlite_FixAuditsInSubmissions(t *testing.T) {
	resetDB(t)

	auditor := &Auditor{
		UserID:   196417,
		Username: "Elly",
		Role:     RoleAuditor,
	}

	err := db.InsertAuditor(*auditor)
	if err != nil {
		t.Fatalf("InsertAuditor() failed: %v", err)
	}

	submission := Submission{
		ID:          123,
		UserID:      456,
		URL:         "url",
		Title:       "title",
		Description: "description",
		Updated:     time.Now(),
	}

	err = db.InsertSubmission(submission)
	if err != nil {
		t.Fatalf("InsertSubmission() failed: %v", err)
	}

	audit := Audit{
		AuditorID:          &auditor.UserID,
		SubmissionID:       submission.ID,
		SubmissionUsername: "User",
		SubmissionUserID:   456,
		Flags:              []Flag{FlagUndisclosed},
		ActionTaken:        "none",
	}

	_, err = db.InsertAudit(audit)
	if err != nil {
		t.Fatalf("InsertAudit() failed: %v", err)
	}

	err = db.FixAuditsInSubmissions()
	if err != nil {
		t.Fatalf("FixAuditsInSubmissions() failed: %v", err)
	}

	submission, err = db.GetSubmissionByID(submission.ID)
	if err != nil {
		t.Fatalf("GetSubmissionByID() failed: %v", err)
	}

	if submission.audit == nil {
		t.Fatalf("FixAuditsInSubmissions() failed: expected non-nil audit, got nil")
	}

	t.Log("TestSqlite_FixAuditsInSubmissions() passed")
}

func TestSqlite_GetAuditByID(t *testing.T) {
	resetDB(t)

	auditor := &Auditor{
		UserID:   196417,
		Username: "Elly",
		Role:     RoleAuditor,
	}

	err := db.InsertAuditor(*auditor)
	if err != nil {
		t.Fatalf("InsertAuditor() failed: %v", err)
	}

	submission := Submission{
		ID:          123,
		UserID:      456,
		URL:         "url",
		Title:       "title",
		Description: "description",
		Updated:     time.Now(),
	}

	err = db.InsertSubmission(submission)
	if err != nil {
		t.Fatalf("InsertSubmission() failed: %v", err)
	}

	audit := Audit{
		AuditorID:          &auditor.UserID,
		SubmissionID:       submission.ID,
		SubmissionUsername: "User",
		SubmissionUserID:   456,
		Flags:              []Flag{FlagUndisclosed},
		ActionTaken:        "none",
	}

	id, err := db.InsertAudit(audit)
	if err != nil {
		t.Fatalf("InsertAudit() failed: %v", err)
	}

	audit, err = db.GetAuditByID(id)
	if err != nil {
		t.Fatalf("GetAuditByID() failed: %v", err)
	}

	if audit.id != id {
		t.Fatalf("GetAuditByID() failed: expected %v, got %v", id, audit.id)
	}

	t.Log("TestSqlite_GetAuditByID() passed")
}

func newPhysical(filename string, t *testing.T) *Sqlite {
	err := touchDBFile(filename)
	if err != nil {
		t.Fatalf("touchDBFile() failed: %v", err)
	}

	db, err := sql.Open("sqlite", filename)
	if err != nil {
		t.Fatalf("sql.Open() failed: %v", err)
	}

	_, err = db.Exec(setForeignKeyCheck)
	if err != nil {
		t.Fatalf("failed to enable foreign key checks: %v", err)
	}

	ctx := context.Background()
	err = migrate(ctx, db)
	if err != nil {
		t.Fatalf("migrate() failed: %v", err)
	}

	return &Sqlite{db, ctx}
}

func TestAllReal(t *testing.T) {
	useVirtualDB = false
	//t.Run("TestPhysical", TestPhysical)
	t.Run("TestNew", TestNew)
	t.Run("TestSqlite_InsertSubmission", TestSqlite_InsertSubmission)
	t.Run("TestSqlite_InsertFile", TestSqlite_InsertFile)
	t.Run("TestSqlite_InsertAuditor", TestSqlite_InsertAuditor)
	t.Run("TestSqlite_InsertModel", TestSqlite_InsertModel)

	t.Run("TestSqlite_InsertSubmission_SQLInjection", TestSqlite_InsertSubmission_SQLInjection)

	t.Run("TestSqlite_InsertFullSubmission", TestSqlite_InsertFullSubmission)
	t.Run("TestSqlite_InsertAudit", TestSqlite_InsertAudit)

	t.Run("TestSqlite_IncreaseAuditCount", TestSqlite_IncreaseAuditCount)
	t.Run("TestSqlite_SyncAuditCount", TestSqlite_SyncAuditCount)
	t.Run("TestSqlite_FixAuditsInSubmissions", TestSqlite_FixAuditsInSubmissions)

	t.Run("TestSqlite_GetSubmissionByID", TestSqlite_GetSubmissionByID)

	t.Run("TestSqlite_GetAuditsByAuditor", TestSqlite_GetAuditsByAuditor)
	t.Run("TestSqlite_GetAuditorByID", TestSqlite_GetAuditorByID)

	t.Run("TestSqlite_GetAuditByID", TestSqlite_GetAuditByID)
	t.Run("TestSqlite_GetAuditBySubmissionID", TestSqlite_GetAuditBySubmissionID)

	t.Run("TestSqlite_ValidSID", TestSqlite_ValidSID)

	t.Run("TestSqlite_InsertTicket", TestSqlite_Tickets)
	useVirtualDB = true
}

func TestSqlite_InsertFullSubmission(t *testing.T) {
	resetDB(t)

	submission := Submission{
		ID:          456,
		UserID:      789,
		URL:         "url",
		Title:       "title",
		Description: "description",
		Updated:     time.Now(),
		AuditID:     nil,
	}

	err := db.InsertSubmission(submission)
	if err != nil {
		t.Fatalf("InsertSubmission() failed: %v", err)
	}

	submission, err = db.GetSubmissionByID(456)
	if err != nil {
		t.Fatalf("GetSubmissionByID() failed: %v", err)
	}

	auditor := &Auditor{UserID: 196417, Username: "Elly", Role: RoleAuditor, AuditCount: 0}

	err = db.InsertAuditor(*auditor)
	if err != nil {
		t.Fatalf("InsertAuditor() failed: %v", err)
	}

	audit := Audit{
		AuditorID:          &auditor.UserID,
		SubmissionID:       456,
		SubmissionUsername: "User",
		SubmissionUserID:   789,
		Flags:              []Flag{FlagUndisclosed},
		ActionTaken:        "none",
	}

	auditID, err := db.InsertAudit(audit)

	submission, err = db.GetSubmissionByID(456)
	if err != nil {
		t.Fatalf("GetSubmissionByID() failed: %v", err)
	}

	if submission.Audit() == nil {
		t.Fatalf("InsertSubmission() failed: expected non-nil audit, got nil")
	}

	if submission.AuditID == nil || *submission.AuditID != auditID {
		t.Fatalf("InsertSubmission() failed: expected %v, got %v", auditID, submission.AuditID)
	}

	if submission.Audit().AuditorID == nil || *submission.Audit().AuditorID != 196417 {
		t.Fatalf("InsertSubmission() failed: expected 196417, got %v", submission.audit.auditor.UserID)
	}

	if submission.Audit().Auditor().UserID != 196417 {
		t.Fatalf("InsertSubmission() failed: expected 196417, got %v", submission.audit.auditor.UserID)
	}

	t.Logf("TestSqlite_GetSubmissionByID2() passed: %+v", submission)
}

func TestSqlite_ValidSID(t *testing.T) {
	resetDB(t)
	user := &api.Credentials{
		UserID:   196417,
		Username: "Elly",
		Sid:      "sid",
	}

	hash := HashCredentials(*user)
	t.Logf("hash: %v", hash)
	err := db.InsertSIDHash(hash)
	if err != nil {
		t.Fatalf("InsertSIDHash() failed: %v", err)
	}

	stored, err := db.GetSIDsFromUserID(196417)
	if err != nil {
		t.Fatalf("GetSIDsFromUserID() failed: %v", err)
	}
	t.Logf("stored: %v", stored)

	if !db.ValidSID(*user) {
		t.Fatalf("ValidSID() failed: expected true, got false")
	}

	if slow {
		user, err = api.Guest().Login()
		if err != nil {
			t.Fatalf("Login() failed: %v", err)
		}
	} else {
		user.Sid = "sid"
	}

	err = db.InsertSIDHash(HashCredentials(*user))
	if err != nil {
		t.Fatalf("InsertSIDHash() failed: %v", err)
	}

	if !db.ValidSID(*user) {
		t.Fatalf("ValidSID() failed: expected true, got false")
	}
}

func TestSqlite_InsertModel(t *testing.T) {
	resetDB(t)
	models := ModelHashes{"18202d0ba2": []string{"furtasticv20_furtasticv20"}}

	err := db.InsertModel(models)
	if err != nil {
		t.Fatalf("InsertModel() failed: %v", err)
	}

	known := db.ModelNamesFromHash("18202d0ba2")
	if len(known) == 0 {
		t.Fatalf("ModelNamesFromHash() failed: expected > 0, got 0")
	}

	if !slices.Contains(known, "furtasticv20_furtasticv20") {
		t.Fatalf("ModelNamesFromHash() failed: expected furtasticv20_furtasticv20, got %v", known[0])
	}
}

func TestSqlite_Tickets(t *testing.T) {
	useVirtualDB = false
	resetDB(t)

	submission := Submission{
		ID:          123,
		UserID:      456,
		URL:         "url",
		Title:       "title",
		Description: "description",
		Generated:   true,
	}

	err := db.InsertSubmission(submission)
	if err != nil {
		t.Fatalf("could not insert submission: %v", err)
	}

	auditor := &Auditor{UserID: 196417, Username: "Elly", Role: RoleAuditor, AuditCount: 0}

	err = db.InsertAuditor(*auditor)
	if err != nil {
		t.Fatalf("could not insert auditor: %v", err)
	}

	ticket := Ticket{
		ID:         1,
		Subject:    "subject",
		DateOpened: time.Now().UTC(),
		Status:     "Open",
		Labels:     []TicketLabel{LabelAIGenerated},
		Priority:   "low",
		Closed:     false,
		Responses: []Response{
			{
				SupportTeam: false,
				Date:        time.Now().UTC(),
				Message:     "The following submission doesn't include the prompts: https://inkbunny.net/s/123",
			},
		},
		SubmissionIDs: []int64{123},
		AssignedID:    &auditor.UserID,
		UsersInvolved: Involved{
			Reporter:    api.UsernameID{UserID: "196417", Username: "Elly"},
			ReportedIDs: []api.UsernameID{{UserID: "456", Username: "User"}},
		},
	}

	err = db.UpsertTicket(ticket)
	if err != nil {
		t.Fatalf("could not insert ticket: %v", err)
	}

	ticket = Ticket{
		ID:         2,
		Subject:    ticket.Subject,
		DateOpened: time.Now().UTC(),
		Status:     "Closed",
		Labels:     []TicketLabel{LabelAIGenerated, LabelAIAssisted},
		Priority:   "high",
		Closed:     true,
		Responses: []Response{
			{
				SupportTeam: false,
				Date:        time.Now().UTC(),
				Message:     "The following submission doesn't include the prompts: https://inkbunny.net/s/123",
			},
		},
		SubmissionIDs: []int64{123},
		AssignedID:    &auditor.UserID,
		UsersInvolved: Involved{
			Reporter:    api.UsernameID{UserID: "196417", Username: "Elly"},
			ReportedIDs: []api.UsernameID{{UserID: "456", Username: "User"}},
		},
	}

	err = db.UpsertTicket(ticket)
	if err != nil {
		t.Fatalf("could not insert ticket: %v", err)
	}

	tickets, err := db.GetTicketsByAuditor(196417)
	if err != nil {
		t.Fatalf("could not get tickets: %v", err)
	}

	if len(tickets) == 0 {
		t.Fatalf("GetTicketsByAuditor() failed: expected > 0, got 0")
	}

	stringFuncs := map[string]func(string) ([]Ticket, error){
		"Open":                   db.GetTicketsByStatus,
		"Closed":                 db.GetTicketsByStatus,
		"low":                    db.GetTicketsByPriority,
		"high":                   db.GetTicketsByPriority,
		string(LabelAIGenerated): db.GetTicketsByLabel,
		string(LabelAIAssisted):  db.GetTicketsByLabel,
	}

	for arg, f := range stringFuncs {
		t.Logf("TestSqlite_Tickets() %v", arg)
		tickets, err = f(arg)
		if err != nil {
			t.Errorf("failed: %v", err)
		}

		if len(tickets) == 0 {
			t.Errorf("failed: expected > 0, got 0: %v", arg)
		}
	}

	tickets, err = db.GetOpenTickets()
	if err != nil {
		t.Fatalf("could not get tickets: %v", err)
	}

	if len(tickets) == 0 {
		t.Fatalf("GetOpenTickets() failed: expected > 0, got 0")
	}

	tickets, err = db.GetClosedTickets()
	if err != nil {
		t.Fatalf("could not get tickets: %v", err)
	}

	if len(tickets) == 0 {
		t.Fatalf("GetClosedTickets() failed: expected > 0, got 0")
	}
}

func TestAssertArgs(t *testing.T) {
	float := float64(1)
	s := "string"
	slice := []string{s}
	m := map[string]string{"key": "value"}
	str := struct{ String string }{String: s}
	strs := []struct{ String string }{{String: s}}

	for i, args := range [][]any{
		{
			float,
			s,
			slice,
			m,
			str,
			strs,
		},
		{
			&float,
			&s,
			&slice,
			&m,
			&str,
			&strs,
		},
		{
			(*float64)(nil),
			(*string)(nil),
			(*[]string)(nil),
			(*map[string]string)(nil),
			(*struct{ String string })(nil),
			(*[]struct{ String string })(nil),
		},
	} {
		t.Logf("TestAssertArgs() %v", i)
		args, err := assertArgs(args...)
		if err != nil {
			t.Fatalf("assertArgs() failed: %v", err)
		}

		if len(args) == 0 {
			t.Fatalf("assertArgs() failed: expected > 0, got 0")
		}
	}
}

func TestScans(t *testing.T) {
	t.Run("TestScan", TestScan)
	t.Run("TestScanPointer", TestScanPointer)
	t.Run("TestScanBoth", TestScanBoth)
	t.Run("TestScanBytes", TestScanBytes)
}

func TestScan(t *testing.T) {
	var fieldsToSet struct {
		Float   float64
		String  string
		Slice   []string
		Map     map[string]string
		Struct  struct{ String string }
		Structs []struct{ String string }
	}

	rows := map[any]any{
		&fieldsToSet.Float:   1.0,
		&fieldsToSet.String:  "string",
		&fieldsToSet.Slice:   []string{"string"},
		&fieldsToSet.Map:     map[string]string{"key": "value"},
		&fieldsToSet.Struct:  struct{ String string }{"string"},
		&fieldsToSet.Structs: []struct{ String string }{{"string"}},
	}

	err := Scan(rows)
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
	}

	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{"Float", fieldsToSet.Float, 1.0},
		{"String", fieldsToSet.String, "string"},
		{"Slice", fieldsToSet.Slice, []string{"string"}},
		{"Map", fieldsToSet.Map, map[string]string{"key": "value"}},
		{"Struct", fieldsToSet.Struct, struct{ String string }{"string"}},
		{"Structs", fieldsToSet.Structs, []struct{ String string }{{"string"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.expected) {
				t.Errorf("Expected %v to be %+v, got %+v", tt.name, tt.expected, tt.got)
			} else {
				t.Logf("TestScan() %v passed, expected %+v, got %+v", tt.name, tt.expected, tt.got)
			}
		})
	}
}

func TestScanPointer(t *testing.T) {
	var fieldsToSet struct {
		Float   *float64
		String  *string
		Slice   *[]string
		Map     *map[string]string
		Struct  *struct{ String string }
		Structs *[]struct{ String string }
	}

	rows := map[any]any{
		&fieldsToSet.Float:   1.0,
		&fieldsToSet.String:  "string",
		&fieldsToSet.Slice:   &[]string{"string"},
		&fieldsToSet.Map:     &map[string]string{"key": "value"},
		&fieldsToSet.Struct:  &struct{ String string }{"string"},
		&fieldsToSet.Structs: &[]struct{ String string }{{"string"}},
	}

	err := Scan(rows)
	if err != nil {
		t.Fatalf("ScanPointer() failed: %v", err)
	}

	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{"Float", *fieldsToSet.Float, 1.0},
		{"String", *fieldsToSet.String, "string"},
		{"Slice", *fieldsToSet.Slice, []string{"string"}},
		{"Map", *fieldsToSet.Map, map[string]string{"key": "value"}},
		{"Struct", *fieldsToSet.Struct, struct{ String string }{"string"}},
		{"Structs", *fieldsToSet.Structs, []struct{ String string }{{"string"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.expected) {
				t.Errorf("Expected %v to be %+v, got %+v", tt.name, tt.expected, tt.got)
			} else {
				t.Logf("TestScanPointer() %v passed, expected %+v, got %+v", tt.name, tt.expected, tt.got)
			}
		})
	}
}

func TestScanBoth(t *testing.T) {
	var fieldsToSet struct {
		Float              float64
		FloatPtr           *float64
		FloatPtrValue      float64
		FloatPtrField      *float64
		String             string
		StringPtr          *string
		StringPtrValue     string
		StringPtrField     *string
		Slice              []string
		SlicePtr           *[]string
		SlicePtrValue      []string
		SlicePtrField      *[]string
		Map                map[string]string
		MapPtr             *map[string]string
		MapPtrValue        map[string]string
		MapPtrField        *map[string]string
		Time               time.Time
		TimePtr            *time.Time
		TimePtrValue       time.Time
		TimePtrField       *time.Time
		TimeString         time.Time
		TimeStringPtr      *time.Time
		TimeStringPtrValue time.Time
		TimeStringPtrField *time.Time
		Nil                *string
	}

	float := float64(1)
	str := "string"
	now := time.Now().UTC()
	formatted := now.Format(time.RFC3339Nano)

	fieldsToSet.Nil = &str

	rows := map[any]any{
		&fieldsToSet.Float:              1.0,
		&fieldsToSet.FloatPtr:           &float,
		&fieldsToSet.FloatPtrValue:      &float,
		&fieldsToSet.FloatPtrField:      1.0,
		&fieldsToSet.String:             "string",
		&fieldsToSet.StringPtr:          &str,
		&fieldsToSet.StringPtrValue:     &str,
		&fieldsToSet.StringPtrField:     "string",
		&fieldsToSet.Slice:              []string{"string"},
		&fieldsToSet.SlicePtr:           &[]string{"string"},
		&fieldsToSet.SlicePtrValue:      &[]string{"string"},
		&fieldsToSet.SlicePtrField:      []string{"string"},
		&fieldsToSet.Map:                map[string]string{"key": "value"},
		&fieldsToSet.MapPtr:             &map[string]string{"key": "value"},
		&fieldsToSet.MapPtrValue:        &map[string]string{"key": "value"},
		&fieldsToSet.MapPtrField:        map[string]string{"key": "value"},
		&fieldsToSet.Time:               now,
		&fieldsToSet.TimePtr:            &now,
		&fieldsToSet.TimePtrValue:       &now,
		&fieldsToSet.TimePtrField:       now,
		&fieldsToSet.TimeString:         formatted,
		&fieldsToSet.TimeStringPtr:      &formatted,
		&fieldsToSet.TimeStringPtrValue: &formatted,
		&fieldsToSet.TimeStringPtrField: formatted,
		&fieldsToSet.Nil:                nil,
	}

	err := Scan(rows)
	if err != nil {
		t.Fatalf("ScanPointer() failed: %v", err)
	}

	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{"Float", fieldsToSet.Float, 1.0},
		{"FloatPtr", *fieldsToSet.FloatPtr, 1.0},
		{"FloatPtrValue", fieldsToSet.FloatPtrValue, 1.0},
		{"FloatPtrField", *fieldsToSet.FloatPtrField, 1.0},
		{"String", fieldsToSet.String, "string"},
		{"StringPtr", *fieldsToSet.StringPtr, "string"},
		{"StringPtrValue", fieldsToSet.StringPtrValue, "string"},
		{"StringPtrField", *fieldsToSet.StringPtrField, "string"},
		{"Slice", fieldsToSet.Slice, []string{"string"}},
		{"SlicePtr", *fieldsToSet.SlicePtr, []string{"string"}},
		{"SlicePtrValue", fieldsToSet.SlicePtrValue, []string{"string"}},
		{"SlicePtrField", *fieldsToSet.SlicePtrField, []string{"string"}},
		{"Map", fieldsToSet.Map, map[string]string{"key": "value"}},
		{"MapPtr", *fieldsToSet.MapPtr, map[string]string{"key": "value"}},
		{"MapPtrValue", fieldsToSet.MapPtrValue, map[string]string{"key": "value"}},
		{"MapPtrField", *fieldsToSet.MapPtrField, map[string]string{"key": "value"}},
		{"Time", fieldsToSet.Time, now},
		{"TimePtr", *fieldsToSet.TimePtr, now},
		{"TimePtrValue", fieldsToSet.TimePtrValue, now},
		{"TimePtrField", *fieldsToSet.TimePtrField, now},
		{"TimeString", fieldsToSet.TimeString, now},
		{"TimeStringPtr", *fieldsToSet.TimeStringPtr, now},
		{"TimeStringPtrValue", fieldsToSet.TimeStringPtrValue, now},
		{"TimeStringPtrField", *fieldsToSet.TimeStringPtrField, now},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.expected) {
				t.Errorf("Expected %v to be %+v, got %+v", tt.name, tt.expected, tt.got)
			} else {
				t.Logf("TestScanPointer() %v passed, expected %+v, got %+v", tt.name, tt.expected, tt.got)
			}
		})
	}

	if fieldsToSet.Nil != nil {
		t.Errorf("Expected nil, got %v", fieldsToSet.Nil)
	}
}

func TestScanBytes(t *testing.T) {
	var fieldsToSet struct {
		Float      float64
		FloatPtr   *float64
		String     string
		StringPtr  *string
		Slice      []string
		SlicePtr   *[]string
		Map        map[string]string
		MapPtr     *map[string]string
		Struct     struct{ String string }
		StructPtr  *struct{ String string }
		Structs    []struct{ String string }
		StructsPtr *[]struct{ String string }
	}

	rows := map[any]any{
		&fieldsToSet.Float:      []byte(`1.0`),
		&fieldsToSet.FloatPtr:   []byte(`1.0`),
		&fieldsToSet.String:     []byte(`"string"`),
		&fieldsToSet.StringPtr:  []byte(`"string"`),
		&fieldsToSet.Slice:      []byte(`["string"]`),
		&fieldsToSet.SlicePtr:   []byte(`["string"]`),
		&fieldsToSet.Map:        []byte(`{"key":"value"}`),
		&fieldsToSet.MapPtr:     []byte(`{"key":"value"}`),
		&fieldsToSet.Struct:     []byte(`{"String":"string"}`),
		&fieldsToSet.StructPtr:  []byte(`{"String":"string"}`),
		&fieldsToSet.Structs:    []byte(`[{"String":"string"}]`),
		&fieldsToSet.StructsPtr: []byte(`[{"String":"string"}]`),
	}

	err := Scan(rows)
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
	}

	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{"Float", fieldsToSet.Float, 1.0},
		{"FloatPtr", *fieldsToSet.FloatPtr, 1.0},
		{"String", fieldsToSet.String, "string"},
		{"StringPtr", *fieldsToSet.StringPtr, "string"},
		{"Slice", fieldsToSet.Slice, []string{"string"}},
		{"SlicePtr", *fieldsToSet.SlicePtr, []string{"string"}},
		{"Map", fieldsToSet.Map, map[string]string{"key": "value"}},
		{"MapPtr", *fieldsToSet.MapPtr, map[string]string{"key": "value"}},
		{"Struct", fieldsToSet.Struct, struct{ String string }{"string"}},
		{"StructPtr", *fieldsToSet.StructPtr, struct{ String string }{"string"}},
		{"Structs", fieldsToSet.Structs, []struct{ String string }{{"string"}}},
		{"StructsPtr", *fieldsToSet.StructsPtr, []struct{ String string }{{"string"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.expected) {
				t.Errorf("Expected %v to be %+v, got %+v", tt.name, tt.expected, tt.got)
			} else {
				t.Logf("TestScan() %v passed, expected %+v, got %+v", tt.name, tt.expected, tt.got)
			}
		})
	}
}
