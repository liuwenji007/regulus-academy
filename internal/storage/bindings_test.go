package storage

import (
	"path/filepath"
	"testing"
)

func TestChannelBindings(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, err := store.CreateUser("小明")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.UpsertChannelBinding(PlatformTelegram, "tg-123", user.ID, user.DisplayName); err != nil {
		t.Fatal(err)
	}

	b, err := store.GetChannelBinding(PlatformTelegram, "tg-123")
	if err != nil || b == nil || b.UserID != user.ID {
		t.Fatalf("binding: %+v err=%v", b, err)
	}

	found, err := store.FindUserByDisplayName("小明")
	if err != nil || found.ID != user.ID {
		t.Fatalf("FindUserByDisplayName: %+v err=%v", found, err)
	}

	if err := store.SetChannelActiveNode(user.ID, "dom-1", "node-1"); err != nil {
		t.Fatal(err)
	}
	active, err := store.GetChannelActiveNode(user.ID)
	if err != nil || active.NodeKey != "node-1" {
		t.Fatalf("active node: %+v err=%v", active, err)
	}
}

const PlatformTelegram = "telegram"
