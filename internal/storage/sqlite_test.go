package storage

import (
	"os"
	"testing"
	"time"
)

func setupTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dbPath := "test_" + t.Name() + ".db"
	t.Cleanup(func() { os.Remove(dbPath) })

	store, err := NewSQLite(dbPath)
	if err != nil {
		t.Fatal("failed to create store:", err)
	}
	t.Cleanup(func() { store.Close() })

	return store
}

func TestSaveAndGet(t *testing.T) {
	store := setupTestStore(t)

	if err := store.Save("abc123", "https://go.dev", nil); err != nil {
		t.Fatal("Save failed:", err)
	}

	got, err := store.Get("abc123")
	if err != nil {
		t.Fatal("Get failed:", err)
	}
	if got != "https://go.dev" {
		t.Errorf("got %q, want %q", got, "https://go.dev")
	}
}

func TestGetNotFound(t *testing.T) {
	store := setupTestStore(t)

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent code, got nil")
	}
}

func TestDuplicateShortCode(t *testing.T) {
	store := setupTestStore(t)

	store.Save("dup123", "https://go.dev", nil)
	err := store.Save("dup123", "https://github.com", nil)
	if err == nil {
		t.Error("expected error for duplicate short code, got nil")
	}
}

func TestIncrementClick(t *testing.T) {
	store := setupTestStore(t)
	store.Save("click1", "https://go.dev", nil)

	for range 5 {
		if err := store.IncrementClick("click1"); err != nil {
			t.Fatal("IncrementClick failed:", err)
		}
	}

	stats, err := store.GetStats("click1")
	if err != nil {
		t.Fatal("GetStats failed:", err)
	}
	if stats.ClickCount != 5 {
		t.Errorf("click_count = %d, want 5", stats.ClickCount)
	}
}

func TestGetStats(t *testing.T) {
	store := setupTestStore(t)
	store.Save("stat1", "https://github.com", nil)

	stats, err := store.GetStats("stat1")
	if err != nil {
		t.Fatal("GetStats failed:", err)
	}
	if stats.ShortCode != "stat1" {
		t.Errorf("short_code = %q, want %q", stats.ShortCode, "stat1")
	}
	if stats.OriginalURL != "https://github.com" {
		t.Errorf("original_url = %q, want %q", stats.OriginalURL, "https://github.com")
	}
	if stats.ClickCount != 0 {
		t.Errorf("click_count = %d, want 0", stats.ClickCount)
	}
}

func TestExpiredURLNotReturned(t *testing.T) {
	store := setupTestStore(t)

	expiry := time.Now().Add(-1 * time.Second)
	store.Save("exp01", "https://go.dev", &expiry)

	_, err := store.Get("exp01")
	if err == nil {
		t.Error("expected error for expired URL, got nil")
	}
}

func TestNonExpiredURLReturned(t *testing.T) {
	store := setupTestStore(t)

	expiry := time.Now().Add(1 * time.Hour)
	store.Save("live1", "https://go.dev", &expiry)

	got, err := store.Get("live1")
	if err != nil {
		t.Fatal("Get failed:", err)
	}
	if got != "https://go.dev" {
		t.Errorf("got %q, want %q", got, "https://go.dev")
	}
}
