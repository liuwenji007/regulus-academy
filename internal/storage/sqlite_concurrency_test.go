package storage

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrentSessionWrites(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, tree, err := store.CreateDomain("并发测试")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := store.CreateSession(DefaultUserID, tree.DomainID, "", "node_a", "exercise", nil)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 24)
	for i := 0; i < 12; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := store.AddMessage(sess.ID, "user", "答案")
			errCh <- err
		}()
		go func() {
			defer wg.Done()
			_, err := store.GetSession(sess.ID)
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent access: %v", err)
		}
	}
}
