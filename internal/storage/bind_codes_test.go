package storage

import (
	"path/filepath"
	"testing"
)

func TestBindCodeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, err := store.CreateUser("测试绑定")
	if err != nil {
		t.Fatal(err)
	}
	code, _, err := store.CreateBindCode(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.RedeemBindCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if got != user.ID {
		t.Fatalf("user id: got %s want %s", got, user.ID)
	}
	_, err = store.RedeemBindCode(code)
	if err == nil {
		t.Fatal("expected error on second redeem")
	}
}
