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

func tempDB(ctx context.Context) (*Sqlite, error) {
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
		UserID:   "196417",
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
		UserID:   "196417",
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
		UserID:   "196417",
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

	if auditor.AuditCount != 0 {
		t.Fatalf("SyncAuditCount() failed: expected 0, got %v", auditor.AuditCount)
	}

	submission := Submission{
		ID:          "123",
		UserID:      "456",
		URL:         "url",
		Title:       "title",
		Description: "description",
		Audit: &Audit{
			Auditor:            auditor,
			SubmissionID:       "123",
			SubmissionUsername: "User",
			SubmissionUserID:   "456",
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

	audits, err := db.GetAuditsByAuditor("196417")
	if err != nil {
		t.Fatalf("GetAuditsByAuditor() failed: %v", err)
	}

	if len(audits) != 0 {
		t.Fatalf("GetAuditsByAuditor() failed: expected 0, got %v", len(audits))
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
		ID:          "123",
		UserID:      "456",
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

	auditor, err := db.GetAuditorByID("196417")
	if err != nil {
		t.Fatalf("GetAuditorByID() failed: %v", err)
	}

	err = db.InsertSubmission(Submission{ID: "123"})
	if err != nil {
		t.Fatalf("InsertSubmission() failed: %v", err)
	}

	audit := Audit{
		Auditor:            auditor,
		SubmissionID:       "123",
		SubmissionUsername: "User",
		SubmissionUserID:   "456",
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
		UserID:   "196417",
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
		ID:          "123",
		UserID:      "456",
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
