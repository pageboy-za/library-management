package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func getAllBooksHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	books, err := getAllBooks(db)
	if err != nil {
		http.Error(w, "Failed to get books: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

func getAllIsbnsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	isbns, err := getAllIsbns(db)
	if err != nil {
		http.Error(w, "Failed to get isbns: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(isbns)
}

func getBookHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, isbn string) {
	book, err := getBook(db, isbn)
	if err == sql.ErrNoRows {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

func submitBookHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	isbn := r.FormValue("isbn")
	title := r.FormValue("title")
	yearStr := r.FormValue("year")
	coverURL := r.FormValue("cover_url")
	authorNames := strings.Split(r.FormValue("authors"), ",")

	var year sql.NullInt64
	if yearStr != "" {
		yearInt, err := strconv.Atoi(yearStr)
		if err != nil {
			http.Error(w, "Invalid year format", http.StatusBadRequest)
			return
		}
		year = sql.NullInt64{Int64: int64(yearInt), Valid: true}
	} else {
		year = sql.NullInt64{Valid: false}
	}

	var coverURLNullable sql.NullString
	if coverURL != "" {
		coverURLNullable = sql.NullString{String: coverURL, Valid: true}
	} else {
		coverURLNullable = sql.NullString{Valid: false}
	}

	if err := insertBookWithAuthors(db, isbn, title, year, coverURLNullable, authorNames); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getAllBooks(db *sql.DB) ([]Book, error) {
	bookQuery := `SELECT
            isbn,
            title,
            subtitle,
            year,
            cover_url,
        FROM
            books 
			id`

	rows, err := db.Query(bookQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book

	return books, nil
}

func getAllIsbns(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT isbn FROM Books")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var isbns []string
	for rows.Next() {
		var isbn string
		if err := rows.Scan(&isbn); err != nil {
			return nil, err
		}
		isbns = append(isbns, isbn)
	}

	return isbns, nil
}

func getBook(db *sql.DB, isbn string) (Book, error) {
	var book Book

	query := `
        SELECT
            b.isbn,
            b.title,
            b.subtitle,
            b.year,
            b.cover_url,
            GROUP_CONCAT(a.name, ', ') AS authors
        FROM
            books b
        JOIN
            BookAuthors ba ON b.id = ba.book_id
        JOIN
            authors a ON ba.author_id = a.id
        WHERE
            b.isbn = ?
        GROUP BY
            b.isbn;
    `
	err := db.QueryRow(query, isbn).Scan(
		// &book.ID,
		&book.ISBN,
		&book.Title,
		&book.Subtitle,
		&book.Year,
		&book.CoverURL,
		&book.Authors,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return book, fmt.Errorf("book not found")
		}
		return book, err
	}

	return book, nil
}

func getAuthorsForBook(db *sql.DB, id string) ([]Author, error) {
	rows, err := db.Query(`
           SELECT a.id, a.name
    FROM authors a
    JOIN BookAuthors ba ON a.id = ba.author_id
    WHERE ba.book_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var authors []Author
	for rows.Next() {
		var author Author
		if err := rows.Scan(&author.ID, &author.Name); err != nil {
			return nil, err
		}
		authors = append(authors, author)
	}

	return authors, nil
}

func insertBookWithAuthors(db *sql.DB, isbn, title string, year sql.NullInt64, coverURL sql.NullString, authorNames []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO books (isbn, title, year, cover_url) VALUES (?, ?, ?, ?)", isbn, title, year, coverURL)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, authorName := range authorNames {
		authorName = strings.TrimSpace(authorName)
		var authorID int
		err = tx.QueryRow("SELECT id FROM authors WHERE name = ?", authorName).Scan(&authorID)
		if err == sql.ErrNoRows {
			res, err := tx.Exec("INSERT INTO authors (name) VALUES (?)", authorName)
			if err != nil {
				tx.Rollback()
				return err
			}
			lastInsertID, _ := res.LastInsertId()
			authorID = int(lastInsertID)
		} else if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec("INSERT INTO book_authors (book_isbn, author_id) VALUES (?, ?)", isbn, authorID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
