package db

import (
	"context"
	"database/sql"
	"errors"
	"github.com/ellypaws/inkbunny/api"
	"testing"
	"time"
)

var db, _ = tempDB(context.Background())
var useVirtualDB = true

const tempPhysicalDB = "temp.sqlite"

func tempDB(ctx context.Context) (*Sqlite, error) {
	if !useVirtualDB {
		db := newPhysical(tempPhysicalDB, nil)
		return db, nil
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(setForeignKeyCheck)
	if err != nil {
		return nil, errors.New("failed to enable foreign key checks")
	}

	err = migrate(ctx, db)
	if err != nil {
		return nil, err
	}

	return &Sqlite{db, ctx}, nil
}

func resetDB(t *testing.T) {
	var err error
	db, err = tempDB(context.Background())
	if err != nil {
		t.Fatalf("tempDB() failed: %v", err)
	}
}

func TestPhysical(t *testing.T) {
	db, err := New(context.Background())
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

	auditor := &Auditor{
		UserID:   196417,
		Username: "Elly",
		Role:     RoleAuditor,
	}

	err := db.InsertAuditor(*auditor)
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

	auditor := &Auditor{
		UserID:   196417,
		Username: "Elly",
		Role:     RoleAuditor,
	}

	err := db.InsertAuditor(*auditor)
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
		if auditor.AuditCount != 1 {
			t.Errorf("SyncAuditCount() failed: expected 1, got %v", auditor.AuditCount)
		}
	}

	submission := Submission{
		ID:          123,
		UserID:      456,
		URL:         "url",
		Title:       "title",
		Description: "description",
		Audit: &Audit{
			Auditor:            auditor,
			SubmissionID:       123,
			SubmissionUsername: "User",
			SubmissionUserID:   456,
			Flags:              []Flag{FlagUndisclosed},
			ActionTaken:        "none",
		},
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

	err = db.SyncAuditCount(auditor.UserID)
	if err != nil {
		t.Fatalf("SyncAuditCount() failed: %v", err)
	}

	auditor, err = db.GetAuditorByID(auditor.UserID)
	if err != nil {
		t.Fatalf("GetAuditorByID() failed: %v", err)
	}

	if auditor.AuditCount != 1 {
		t.Fatalf("SyncAuditCount() failed: expected 1, got %v", auditor.AuditCount)
	}

	t.Log("TestSqlite_SyncAuditCount() passed")
}

func TestSqlite_GetAuditsByAuditor(t *testing.T) {
	resetDB(t)

	audits, err := db.GetAuditsByAuditor(196417)
	if err != nil {
		t.Fatalf("GetAuditsByAuditor() failed: %v", err)
	}

	if useVirtualDB {
		if len(audits) != 0 {
			t.Fatalf("GetAuditsByAuditor() failed: expected 0, got %v", len(audits))
		}
	} else {
		if len(audits) != 1 {
			t.Fatalf("GetAuditsByAuditor() failed: expected 1, got %v", len(audits))
		}
	}

	t.Log("TestSqlite_GetAuditsByAuditor() passed")
}

func TestSqlite_InsertFile(t *testing.T) {
	resetDB(t)

	file := File{
		File: api.File{
			FileID:   "123",
			FileName: "file",
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
		Auditor:            auditor,
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

	auditor := &Auditor{
		UserID:   196417,
		Username: "Elly",
		Role:     RoleAuditor,
	}

	err := db.InsertAuditor(*auditor)
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
		Auditor:            auditor,
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

	if submission.Audit == nil {
		t.Fatalf("GetAuditBySubmissionID() failed: expected non-nil, got nil")
	}

	if submission.Audit.ID != id {
		t.Fatalf("GetAuditBySubmissionID() failed: expected %v, got %v", audit.ID, submission.Audit.ID)
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
		t.Fatalf("GetSubmissions() failed: %v", err)
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
		Auditor:            auditor,
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

	if submission.Audit == nil {
		t.Fatalf("FixAuditsInSubmissions() failed: expected non-nil, got nil")
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
		Auditor:            auditor,
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

	if audit.ID != id {
		t.Fatalf("GetAuditByID() failed: expected %v, got %v", id, audit.ID)
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
	t.Run("TestSqlite_InsertAuditor", TestSqlite_InsertAuditor)
	t.Run("TestSqlite_IncreaseAuditCount", TestSqlite_IncreaseAuditCount)
	t.Run("TestSqlite_InsertFile", TestSqlite_InsertFile)
	t.Run("TestSqlite_InsertSubmission", TestSqlite_InsertSubmission)
	t.Run("TestSqlite_InsertAudit", TestSqlite_InsertAudit)
	t.Run("TestSqlite_SyncAuditCount", TestSqlite_SyncAuditCount)
	t.Run("TestSqlite_FixAuditsInSubmissions", TestSqlite_FixAuditsInSubmissions)
	t.Run("TestSqlite_GetAuditsByAuditor", TestSqlite_GetAuditsByAuditor)
	t.Run("TestSqlite_GetAuditorByID", TestSqlite_GetAuditorByID)
	t.Run("TestSqlite_GetAuditBySubmissionID", TestSqlite_GetAuditBySubmissionID)
	t.Run("TestSqlite_GetSubmissionByID", TestSqlite_GetSubmissionByID)
	t.Run("TestSqlite_InsertSubmission_SQLInjection", TestSqlite_InsertSubmission_SQLInjection)
	useVirtualDB = true
}
