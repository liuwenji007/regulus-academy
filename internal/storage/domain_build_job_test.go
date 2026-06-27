package storage

import (
	"path/filepath"
	"testing"
)

func TestGetDomainBuildJob_readsJobKindAndDomainID(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	const domainID = "domain-audit-123"
	job, err := store.CreateDomainBuildJobEx(DefaultUserID, "Go 并发", "", false, DomainJobKindAudit, domainID)
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.GetDomainBuildJob(DefaultUserID, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.JobKind != DomainJobKindAudit {
		t.Fatalf("JobKind=%q want %q", got.JobKind, DomainJobKindAudit)
	}
	if got.DomainID != domainID {
		t.Fatalf("DomainID=%q want %q", got.DomainID, domainID)
	}
}

func TestGetDomainBuildJob_defaultsJobKindWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	job, err := store.CreateDomainBuildJob(DefaultUserID, "Rust", "", false)
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.GetDomainBuildJob(DefaultUserID, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.JobKind != DomainJobKindBuild {
		t.Fatalf("JobKind=%q want %q", got.JobKind, DomainJobKindBuild)
	}
	if got.DomainID != "" {
		t.Fatalf("DomainID=%q want empty", got.DomainID)
	}
}
