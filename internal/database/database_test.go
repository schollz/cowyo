package database

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func TestConfigSelectsDatabaseBackend(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   Backend
	}{
		{
			name:   "PostgreSQL when configured",
			config: Config{DatabaseURL: "postgres://localhost/cowyo2"},
			want:   BackendPostgreSQL,
		},
		{
			name:   "SQLite fallback",
			config: Config{},
			want:   BackendSQLite,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.Backend(); got != tt.want {
				t.Fatalf("Backend() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", " postgres://user:secret@localhost/cowyo2 ")
	t.Setenv("SQLITE_PATH", " local.sqlite3 ")

	config := ConfigFromEnv()
	if got := config.DatabaseURL; got != "postgres://user:secret@localhost/cowyo2" {
		t.Errorf("DatabaseURL = %q", got)
	}
	if got := config.SQLitePath; got != "local.sqlite3" {
		t.Errorf("SQLitePath = %q", got)
	}
}

func TestSQLiteStoreMigratesAndPersistsPages(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "cowyo2.sqlite3")
	config := Config{SQLitePath: path}

	store, err := Open(ctx, config)
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}

	if got := store.Backend(); got != BackendSQLite {
		t.Errorf("Backend() = %q, want %q", got, BackendSQLite)
	}

	if _, err := store.GetPage(ctx, "missing"); !errors.Is(err, ErrPageNotFound) {
		t.Fatalf("GetPage() error = %v, want ErrPageNotFound", err)
	}

	random := Page{Title: "random", Text: "first"}
	created, err := store.CreatePage(ctx, random)
	if err != nil {
		t.Fatalf("CreatePage() error = %v", err)
	}
	if !created {
		t.Fatal("CreatePage() = false, want true for a new title")
	}
	created, err = store.CreatePage(ctx, Page{Title: random.Title, Text: "second"})
	if err != nil {
		t.Fatalf("second CreatePage() error = %v", err)
	}
	if created {
		t.Fatal("second CreatePage() = true, want false for an existing title")
	}
	gotRandom, err := store.GetPage(ctx, random.Title)
	if err != nil {
		t.Fatalf("GetPage(random) error = %v", err)
	}
	if gotRandom != random {
		t.Errorf("GetPage(random) = %+v, want original %+v", gotRandom, random)
	}

	want := Page{
		Title:        "test",
		Text:         "hello\nworld",
		CursorStart:  4,
		CursorEnd:    7,
		Published:    true,
		Locked:       true,
		LockSalt:     "stored salt",
		LockVerifier: "stored verifier",
	}
	if err := store.UpsertPage(ctx, want); err != nil {
		t.Fatalf("UpsertPage() error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close SQLite store: %v", err)
	}

	store, err = Open(ctx, config)
	if err != nil {
		t.Fatalf("reopen SQLite store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close reopened SQLite store: %v", err)
		}
	})

	got, err := store.GetPage(ctx, want.Title)
	if err != nil {
		t.Fatalf("GetPage() error = %v", err)
	}
	if got != want {
		t.Errorf("GetPage() = %+v, want %+v", got, want)
	}

	publishedTitles, err := store.ListPublishedPageTitles(ctx)
	if err != nil {
		t.Fatalf("ListPublishedPageTitles() error = %v", err)
	}
	if len(publishedTitles) != 1 || publishedTitles[0] != want.Title {
		t.Errorf("ListPublishedPageTitles() = %v, want [%s]", publishedTitles, want.Title)
	}

	armed := Page{
		Title:        "one-time",
		Text:         "read once",
		Published:    true,
		SelfDestruct: true,
	}
	if err := store.UpsertPage(ctx, armed); err != nil {
		t.Fatalf("store armed page: %v", err)
	}
	publishedTitles, err = store.ListPublishedPageTitles(ctx)
	if err != nil {
		t.Fatalf("list pages after arming self destruct: %v", err)
	}
	if len(publishedTitles) != 1 || publishedTitles[0] != want.Title {
		t.Errorf("published titles include armed page: %v", publishedTitles)
	}

	consumed, err := store.ConsumePage(ctx, armed.Title)
	if err != nil {
		t.Fatalf("ConsumePage(armed) error = %v", err)
	}
	if consumed != armed {
		t.Errorf("ConsumePage(armed) = %+v, want %+v", consumed, armed)
	}
	if _, err := store.GetPage(ctx, armed.Title); !errors.Is(err, ErrPageNotFound) {
		t.Errorf("armed page remains after consumption; GetPage() error = %v", err)
	}

	loaded, err := store.ConsumePage(ctx, want.Title)
	if err != nil {
		t.Fatalf("ConsumePage(unarmed) error = %v", err)
	}
	if loaded != want {
		t.Errorf("ConsumePage(unarmed) = %+v, want %+v", loaded, want)
	}
	if _, err := store.GetPage(ctx, want.Title); err != nil {
		t.Errorf("ConsumePage removed an unarmed page: %v", err)
	}

	lockedArmed := Page{
		Title:        "locked-one-time",
		Text:         "protected",
		SelfDestruct: true,
		Locked:       true,
		LockSalt:     "salt",
		LockVerifier: "verifier",
	}
	if err := store.UpsertPage(ctx, lockedArmed); err != nil {
		t.Fatalf("store locked armed page: %v", err)
	}
	if _, err := store.ConsumePage(ctx, lockedArmed.Title); err != nil {
		t.Fatalf("ConsumePage(locked armed) error = %v", err)
	}
	if _, err := store.GetPage(ctx, lockedArmed.Title); err != nil {
		t.Errorf("ConsumePage removed a locked page: %v", err)
	}

	concurrent := Page{
		Title:        "concurrent-one-time",
		Text:         "only one reader",
		SelfDestruct: true,
	}
	if err := store.UpsertPage(ctx, concurrent); err != nil {
		t.Fatalf("store concurrent armed page: %v", err)
	}

	const readers = 8
	results := make(chan error, readers)
	var readersDone sync.WaitGroup
	for range readers {
		readersDone.Add(1)
		go func() {
			defer readersDone.Done()
			_, err := store.ConsumePage(ctx, concurrent.Title)
			results <- err
		}()
	}
	readersDone.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrPageNotFound) {
			t.Errorf("concurrent ConsumePage() error = %v", err)
		}
	}
	if successes != 1 {
		t.Errorf("concurrent ConsumePage() successes = %d, want 1", successes)
	}
}
