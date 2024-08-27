package main

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Author struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Book struct {
	ISBN     string         `json:"isbn"`
	Title    string         `json:"title"`
	Subtitle sql.NullString `json:"subtitle"`
	Year     sql.NullInt64  `json:"year"`
	ImgURL   sql.NullString `json:"imgURL"`
	Authors  []Author       `json:"authors"`
}

func (b Book) MarshalJSON() ([]byte, error) {
	type Alias Book
	return json.Marshal(&struct {
		Subtitle *string `json:"subtitle,omitempty"`
		Year     *int64  `json:"year,omitempty"`
		ImgURL   *string `json:"imgURL,omitempty"`
		Alias
	}{
		Subtitle: ifNullString(b.Subtitle),
		Year:     ifNullInt64(b.Year),
		ImgURL:   ifNullString(b.ImgURL),
		Alias:    (Alias)(b),
	})
}

func ifNullString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func ifNullInt64(ni sql.NullInt64) *int64 {
	if ni.Valid {
		return &ni.Int64
	}
	return nil
}

func InitializeDatabase(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("error creating tables: %v", err)
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	createBooksTable := `CREATE TABLE IF NOT EXISTS books (
        isbn TEXT PRIMARY KEY,
        title TEXT NOT NULL,
        subtitle TEXT,
        year INTEGER,
        imgURL TEXT
    );`

	createAuthorsTable := `CREATE TABLE IF NOT EXISTS authors (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL
    );`

	createBookAuthorsTable := `CREATE TABLE IF NOT EXISTS book_authors (
        book_isbn TEXT NOT NULL,
        author_id INTEGER NOT NULL,
        FOREIGN KEY (book_isbn) REFERENCES books(isbn),
        FOREIGN KEY (author_id) REFERENCES authors(id),
        PRIMARY KEY (book_isbn, author_id)
    );`

	_, err := db.Exec(createBooksTable)
	if err != nil {
		return fmt.Errorf("error creating books table: %v", err)
	}

	_, err = db.Exec(createAuthorsTable)
	if err != nil {
		return fmt.Errorf("error creating authors table: %v", err)
	}

	_, err = db.Exec(createBookAuthorsTable)
	if err != nil {
		return fmt.Errorf("error creating book_authors table: %v", err)
	}

	return nil
}
