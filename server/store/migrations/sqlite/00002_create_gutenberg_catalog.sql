-- +goose Up
CREATE TABLE IF NOT EXISTS project_gutenberg_catalog (
    id          text PRIMARY KEY,
    title       text NOT NULL,
    authors     text NOT NULL DEFAULT '',
    subjects    text NOT NULL DEFAULT '',
    bookshelves text NOT NULL DEFAULT '',
    languages   text NOT NULL DEFAULT '',
    issued_year integer,
    downloads   integer NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_project_gutenberg_title ON project_gutenberg_catalog(title);
CREATE TABLE IF NOT EXISTS project_gutenberg_catalog_sync (
    id        integer PRIMARY KEY CHECK (id = 1),
    synced_at integer NOT NULL
);

-- +goose Down
DROP TABLE project_gutenberg_catalog_sync;
DROP TABLE project_gutenberg_catalog;
