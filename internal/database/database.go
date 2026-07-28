package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/cowyo2/internal/database/postgresdb"
	"github.com/schollz/cowyo2/internal/database/sqlitedb"
)

const DefaultSQLitePath = "cowyo2.sqlite3"

var ErrPageNotFound = errors.New("page not found")

type Backend string

const (
	BackendPostgreSQL Backend = "postgresql"
	BackendSQLite     Backend = "sqlite"
)

type Config struct {
	DatabaseURL string
	SQLitePath  string
}

func ConfigFromEnv() Config {
	return Config{
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		SQLitePath:  strings.TrimSpace(os.Getenv("SQLITE_PATH")),
	}
}

func (c Config) Backend() Backend {
	if strings.TrimSpace(c.DatabaseURL) != "" {
		return BackendPostgreSQL
	}
	return BackendSQLite
}

type Page struct {
	Title        string
	Text         string
	CursorStart  int64
	CursorEnd    int64
	Published    bool
	SelfDestruct bool
	Locked       bool
	LockSalt     string
	LockVerifier string
}

type pageQueries interface {
	ConsumeSelfDestructPage(context.Context, string) (Page, error)
	CreatePage(context.Context, Page) (bool, error)
	GetPage(context.Context, string) (Page, error)
	ListPublishedPageTitles(context.Context) ([]string, error)
	UpsertPage(context.Context, Page) error
}

type Store struct {
	db      *sql.DB
	backend Backend
	pages   pageQueries
}

func Open(ctx context.Context, config Config) (*Store, error) {
	switch config.Backend() {
	case BackendPostgreSQL:
		return openPostgreSQL(ctx, config.DatabaseURL)
	default:
		return openSQLite(ctx, config.SQLitePath)
	}
}

func openPostgreSQL(ctx context.Context, databaseURL string) (*Store, error) {
	if err := migrateSchema(BackendPostgreSQL, databaseURL); err != nil {
		return nil, fmt.Errorf("migrate PostgreSQL: %w", err)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	return &Store{
		db:      db,
		backend: BackendPostgreSQL,
		pages:   postgresPageQueries{queries: postgresdb.New(db)},
	}, nil
}

func openSQLite(ctx context.Context, path string) (*Store, error) {
	dsn, err := sqliteDataSource(path)
	if err != nil {
		return nil, fmt.Errorf("configure SQLite: %w", err)
	}
	if err := migrateSchema(BackendSQLite, dsn); err != nil {
		return nil, fmt.Errorf("migrate SQLite: %w", err)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to SQLite: %w", err)
	}

	return &Store{
		db:      db,
		backend: BackendSQLite,
		pages:   sqlitePageQueries{queries: sqlitedb.New(db)},
	}, nil
}

func sqliteDataSource(path string) (string, error) {
	if path == "" {
		path = DefaultSQLitePath
	}

	var databaseURL *url.URL
	var err error
	if strings.HasPrefix(path, "file:") {
		databaseURL, err = url.Parse(path)
		if err != nil {
			return "", err
		}
	} else {
		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		databaseURL = &url.URL{
			Scheme: "file",
			Path:   filepath.ToSlash(absolutePath),
		}
	}

	query := databaseURL.Query()
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "foreign_keys(ON)")
	query.Add("_pragma", "journal_mode(WAL)")
	databaseURL.RawQuery = query.Encode()
	return databaseURL.String(), nil
}

func (s *Store) Backend() Backend {
	return s.backend
}

func (s *Store) GetPage(ctx context.Context, title string) (Page, error) {
	return s.pages.GetPage(ctx, title)
}

// ConsumePage atomically removes and returns an armed self-destruct page.
// Pages that are not armed are returned without being modified.
func (s *Store) ConsumePage(ctx context.Context, title string) (Page, error) {
	page, err := s.pages.ConsumeSelfDestructPage(ctx, title)
	if err == nil {
		return page, nil
	}
	if !errors.Is(err, ErrPageNotFound) {
		return Page{}, err
	}
	return s.pages.GetPage(ctx, title)
}

func (s *Store) CreatePage(ctx context.Context, page Page) (bool, error) {
	return s.pages.CreatePage(ctx, page)
}

func (s *Store) ListPublishedPageTitles(ctx context.Context) ([]string, error) {
	return s.pages.ListPublishedPageTitles(ctx)
}

func (s *Store) UpsertPage(ctx context.Context, page Page) error {
	return s.pages.UpsertPage(ctx, page)
}

func (s *Store) Close() error {
	return s.db.Close()
}

type postgresPageQueries struct {
	queries *postgresdb.Queries
}

func (p postgresPageQueries) CreatePage(ctx context.Context, page Page) (bool, error) {
	rows, err := p.queries.CreatePage(ctx, postgresdb.CreatePageParams{
		Title:        page.Title,
		Text:         page.Text,
		CursorStart:  page.CursorStart,
		CursorEnd:    page.CursorEnd,
		Published:    page.Published,
		SelfDestruct: page.SelfDestruct,
		Locked:       page.Locked,
		LockSalt:     page.LockSalt,
		LockVerifier: page.LockVerifier,
	})
	return rows == 1, err
}

func (p postgresPageQueries) ConsumeSelfDestructPage(ctx context.Context, title string) (Page, error) {
	row, err := p.queries.ConsumeSelfDestructPage(ctx, title)
	if errors.Is(err, sql.ErrNoRows) {
		return Page{}, ErrPageNotFound
	}
	if err != nil {
		return Page{}, err
	}
	return Page{
		Title:        row.Title,
		Text:         row.Text,
		CursorStart:  row.CursorStart,
		CursorEnd:    row.CursorEnd,
		Published:    row.Published,
		SelfDestruct: row.SelfDestruct,
		Locked:       row.Locked,
		LockSalt:     row.LockSalt,
		LockVerifier: row.LockVerifier,
	}, nil
}

func (p postgresPageQueries) GetPage(ctx context.Context, title string) (Page, error) {
	row, err := p.queries.GetPage(ctx, title)
	if errors.Is(err, sql.ErrNoRows) {
		return Page{}, ErrPageNotFound
	}
	if err != nil {
		return Page{}, err
	}
	return Page{
		Title:        row.Title,
		Text:         row.Text,
		CursorStart:  row.CursorStart,
		CursorEnd:    row.CursorEnd,
		Published:    row.Published,
		SelfDestruct: row.SelfDestruct,
		Locked:       row.Locked,
		LockSalt:     row.LockSalt,
		LockVerifier: row.LockVerifier,
	}, nil
}

func (p postgresPageQueries) ListPublishedPageTitles(ctx context.Context) ([]string, error) {
	return p.queries.ListPublishedPageTitles(ctx)
}

func (p postgresPageQueries) UpsertPage(ctx context.Context, page Page) error {
	return p.queries.UpsertPage(ctx, postgresdb.UpsertPageParams{
		Title:        page.Title,
		Text:         page.Text,
		CursorStart:  page.CursorStart,
		CursorEnd:    page.CursorEnd,
		Published:    page.Published,
		SelfDestruct: page.SelfDestruct,
		Locked:       page.Locked,
		LockSalt:     page.LockSalt,
		LockVerifier: page.LockVerifier,
	})
}

type sqlitePageQueries struct {
	queries *sqlitedb.Queries
}

func (s sqlitePageQueries) CreatePage(ctx context.Context, page Page) (bool, error) {
	rows, err := s.queries.CreatePage(ctx, sqlitedb.CreatePageParams{
		Title:        page.Title,
		Text:         page.Text,
		CursorStart:  page.CursorStart,
		CursorEnd:    page.CursorEnd,
		Published:    page.Published,
		SelfDestruct: page.SelfDestruct,
		Locked:       page.Locked,
		LockSalt:     page.LockSalt,
		LockVerifier: page.LockVerifier,
	})
	return rows == 1, err
}

func (s sqlitePageQueries) ConsumeSelfDestructPage(ctx context.Context, title string) (Page, error) {
	row, err := s.queries.ConsumeSelfDestructPage(ctx, title)
	if errors.Is(err, sql.ErrNoRows) {
		return Page{}, ErrPageNotFound
	}
	if err != nil {
		return Page{}, err
	}
	return Page{
		Title:        row.Title,
		Text:         row.Text,
		CursorStart:  row.CursorStart,
		CursorEnd:    row.CursorEnd,
		Published:    row.Published,
		SelfDestruct: row.SelfDestruct,
		Locked:       row.Locked,
		LockSalt:     row.LockSalt,
		LockVerifier: row.LockVerifier,
	}, nil
}

func (s sqlitePageQueries) GetPage(ctx context.Context, title string) (Page, error) {
	row, err := s.queries.GetPage(ctx, title)
	if errors.Is(err, sql.ErrNoRows) {
		return Page{}, ErrPageNotFound
	}
	if err != nil {
		return Page{}, err
	}
	return Page{
		Title:        row.Title,
		Text:         row.Text,
		CursorStart:  row.CursorStart,
		CursorEnd:    row.CursorEnd,
		Published:    row.Published,
		SelfDestruct: row.SelfDestruct,
		Locked:       row.Locked,
		LockSalt:     row.LockSalt,
		LockVerifier: row.LockVerifier,
	}, nil
}

func (s sqlitePageQueries) ListPublishedPageTitles(ctx context.Context) ([]string, error) {
	return s.queries.ListPublishedPageTitles(ctx)
}

func (s sqlitePageQueries) UpsertPage(ctx context.Context, page Page) error {
	return s.queries.UpsertPage(ctx, sqlitedb.UpsertPageParams{
		Title:        page.Title,
		Text:         page.Text,
		CursorStart:  page.CursorStart,
		CursorEnd:    page.CursorEnd,
		Published:    page.Published,
		SelfDestruct: page.SelfDestruct,
		Locked:       page.Locked,
		LockSalt:     page.LockSalt,
		LockVerifier: page.LockVerifier,
	})
}
