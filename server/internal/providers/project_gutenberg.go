package providers

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Chandra179/lux/server/internal/books"
)

type projectGutenbergProvider struct {
	db          *sql.DB
	client      *http.Client
	feedURL     string
	initMu      sync.Mutex
	schemaReady bool
}

func newProjectGutenbergProvider(db *sql.DB, client *http.Client, feedURL string) *projectGutenbergProvider {
	return &projectGutenbergProvider{db: db, client: client, feedURL: feedURL}
}

func (p *projectGutenbergProvider) Capabilities() books.ProviderCapability {
	return books.ProviderCapability{ID: "project_gutenberg", Name: "Project Gutenberg", Filters: []string{books.FilterTopics, books.FilterGenre, books.FilterYear, books.FilterLanguage, books.FilterPopularity}, PopularityMetric: "downloads", Notes: []string{"Year is the Project Gutenberg release year, not the original print publication year."}}
}

func (p *projectGutenbergProvider) Search(ctx context.Context, request books.SearchRequest) ([]books.Book, error) {
	if err := p.ensureSchema(ctx); err != nil {
		return nil, err
	}
	terms := queryTerms(request)
	if len(terms) == 0 {
		if strings.EqualFold(strings.ReplaceAll(request.Genre, "-", " "), "non fiction") {
			terms = []string{"history", "science", "psychology", "philosophy", "biography", "economics", "business", "politics", "sociology", "education", "travel", "health", "technology", "religion"}
		} else if strings.TrimSpace(request.Genre) != "" {
			terms = []string{request.Genre}
		} else {
			terms = []string{"non-fiction"}
		}
	}
	conditions := make([]string, 0, len(terms)*3)
	args := make([]any, 0, len(terms)*3+3)
	for _, term := range terms {
		term = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(strings.ToLower(term))
		pattern := "%" + term + "%"
		conditions = append(conditions, "(LOWER(title) LIKE ? ESCAPE '!' OR LOWER(subjects) LIKE ? ESCAPE '!' OR LOWER(bookshelves) LIKE ? ESCAPE '!')")
		args = append(args, pattern, pattern, pattern)
	}
	query := `SELECT id, title, authors, subjects, bookshelves, languages, issued_year, downloads FROM project_gutenberg_catalog WHERE (` + strings.Join(conditions, " OR ") + ")"
	if request.MinYear != nil {
		query += " AND issued_year >= ?"
		args = append(args, *request.MinYear)
	}
	if request.MaxYear != nil {
		query += " AND issued_year <= ?"
		args = append(args, *request.MaxYear)
	}
	if request.Language != "" {
		query += " AND LOWER(languages) LIKE ?"
		args = append(args, "%"+strings.ToLower(request.Language)+"%")
	}
	query += " ORDER BY downloads DESC, title COLLATE NOCASE LIMIT ?"
	args = append(args, withLimit(request, 100))
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query Project Gutenberg catalog: %w", err)
	}
	defer rows.Close()
	result := make([]books.Book, 0)
	for rows.Next() {
		var id, title, authors, subjects, bookshelves, languages string
		var year sql.NullInt64
		var downloads sql.NullInt64
		if err := rows.Scan(&id, &title, &authors, &subjects, &bookshelves, &languages, &year, &downloads); err != nil {
			return nil, err
		}
		book := books.Book{ID: id, Title: title, Authors: splitList(authors), Subjects: append(splitList(subjects), splitList(bookshelves)...), Languages: splitList(languages), URL: "https://www.gutenberg.org/ebooks/" + url.PathEscape(id), YearKind: "gutenberg_release_year"}
		book.Genres = append([]string(nil), book.Subjects...)
		if year.Valid {
			parsed := int(year.Int64)
			book.Year = &parsed
		}
		if downloads.Valid && downloads.Int64 >= 0 {
			book.Popularity = &books.Metric{Value: float64(downloads.Int64), Metric: "downloads"}
		}
		book.Source = books.BookSource{Provider: "project_gutenberg", ID: id, URL: book.URL}
		result = append(result, book)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *projectGutenbergProvider) refreshIfNeeded(ctx context.Context) error {
	if err := p.ensureSchema(ctx); err != nil {
		return err
	}
	var syncedAt sql.NullInt64
	if err := p.db.QueryRowContext(ctx, `SELECT synced_at FROM project_gutenberg_catalog_sync WHERE id = 1`).Scan(&syncedAt); err != nil && err != sql.ErrNoRows {
		return err
	}
	if syncedAt.Valid && time.Since(time.Unix(syncedAt.Int64, 0)) < 24*time.Hour {
		return nil
	}
	return p.refresh(ctx)
}

func (p *projectGutenbergProvider) refresh(ctx context.Context) error {
	if err := p.ensureSchema(ctx); err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.feedURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "text/csv, application/gzip, application/octet-stream")
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("User-Agent", "TrafaeBookDiscovery/1.0")
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("request Project Gutenberg catalog: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Project Gutenberg catalog returned HTTP %d", response.StatusCode)
	}
	var reader io.Reader = response.Body
	if strings.HasSuffix(strings.ToLower(p.feedURL), ".gz") {
		gz, err := gzip.NewReader(response.Body)
		if err != nil {
			return fmt.Errorf("open Project Gutenberg catalog gzip: %w", err)
		}
		defer gz.Close()
		reader = gz
	}
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	header, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("read Project Gutenberg catalog header: %w", err)
	}
	columns := make(map[string]int, len(header))
	for i, field := range header {
		columns[normalizeHeader(field)] = i
	}
	idColumn := column(columns, "text#", "text", "ebook#", "ebook")
	titleColumn := column(columns, "title")
	if idColumn < 0 || titleColumn < 0 {
		return fmt.Errorf("Project Gutenberg CSV is missing required id/title columns")
	}
	rows, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Rollback() }()
	if _, err := rows.ExecContext(ctx, `DELETE FROM project_gutenberg_catalog`); err != nil {
		return err
	}
	statement, err := rows.PrepareContext(ctx, `INSERT OR REPLACE INTO project_gutenberg_catalog(id,title,authors,subjects,bookshelves,languages,issued_year,downloads) VALUES(?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	count := 0
	for {
		record, readErr := csvReader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read Project Gutenberg catalog row: %w", readErr)
		}
		id := valueAt(record, idColumn)
		title := valueAt(record, titleColumn)
		if id == "" || title == "" {
			continue
		}
		year := parseYear(valueAt(record, column(columns, "issued", "release date", "date")))
		var yearValue any
		if year != nil {
			yearValue = *year
		}
		downloads, _ := strconv.ParseInt(valueAt(record, column(columns, "downloads", "download count")), 10, 64)
		if _, err := statement.ExecContext(ctx, id, title, joinList(fieldValues(record, columns, "authors", "author")), joinList(fieldValues(record, columns, "subjects", "subject")), joinList(fieldValues(record, columns, "bookshelves", "shelves")), joinList(fieldValues(record, columns, "language", "languages")), yearValue, downloads); err != nil {
			return fmt.Errorf("store Project Gutenberg row %s: %w", id, err)
		}
		count++
	}
	if count == 0 {
		return fmt.Errorf("Project Gutenberg catalog contained no books")
	}
	if _, err := rows.ExecContext(ctx, `INSERT OR REPLACE INTO project_gutenberg_catalog_sync(id,synced_at) VALUES(1,?)`, time.Now().Unix()); err != nil {
		return err
	}
	if err := rows.Commit(); err != nil {
		return fmt.Errorf("commit Project Gutenberg catalog: %w", err)
	}
	return nil
}

// ensureSchema creates the catalog tables once. A failed attempt is not
// remembered, so a transient error (for example a busy database at startup)
// does not disable the provider for the lifetime of the process.
func (p *projectGutenbergProvider) ensureSchema(ctx context.Context) error {
	p.initMu.Lock()
	defer p.initMu.Unlock()
	if p.schemaReady {
		return nil
	}
	if _, err := p.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS project_gutenberg_catalog (
		id TEXT PRIMARY KEY, title TEXT NOT NULL, authors TEXT NOT NULL DEFAULT '', subjects TEXT NOT NULL DEFAULT '',
		bookshelves TEXT NOT NULL DEFAULT '', languages TEXT NOT NULL DEFAULT '', issued_year INTEGER NULL, downloads INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		return err
	}
	if _, err := p.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_project_gutenberg_title ON project_gutenberg_catalog(title)`); err != nil {
		return err
	}
	if _, err := p.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS project_gutenberg_catalog_sync (id INTEGER PRIMARY KEY CHECK(id = 1), synced_at INTEGER NOT NULL)`); err != nil {
		return err
	}
	p.schemaReady = true
	return nil
}

func normalizeHeader(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "\ufeff")))
}
func column(columns map[string]int, names ...string) int {
	for _, name := range names {
		if index, ok := columns[normalizeHeader(name)]; ok {
			return index
		}
	}
	return -1
}
func valueAt(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}
func fieldValues(record []string, columns map[string]int, names ...string) []string {
	for _, name := range names {
		if value := valueAt(record, column(columns, name)); value != "" {
			return splitList(value)
		}
	}
	return nil
}
func joinList(values []string) string { return strings.Join(values, "\x1f") }
func splitList(value string) []string {
	result := []string{}
	value = strings.ReplaceAll(value, "\x1f", ";")
	for _, item := range strings.Split(value, ";") {
		if strings.TrimSpace(item) != "" {
			result = append(result, strings.TrimSpace(item))
		}
	}
	return result
}
