package db

import (
	"context"
	"testing"
)

var db, _ = New(context.Background())

func TestNew(t *testing.T) {
	ctx := context.Background()
	db, err := New(ctx)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if db == nil {
		t.Fatal("New() failed: db is nil")
	}

	err = db.PingContext(ctx)
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

	t.Log("TestSqlite_SyncAuditCount() passed")
}

func TestSqlite_GetAuditsByAuditor(t *testing.T) {
	audits, err := db.GetAuditsByAuditor("196417")
	if err != nil {
		t.Fatalf("GetAuditsByAuditor() failed: %v", err)
	}

	if len(audits) != 0 {
		t.Fatalf("GetAuditsByAuditor() failed: expected 0, got %v", len(audits))
	}

	t.Log("TestSqlite_GetAuditsByAuditor() passed")
}
