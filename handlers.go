package main

import (
	"database/sql"
	"encoding/json"
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
	imgURL := r.FormValue("imgURL")
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

	var imgURLNullable sql.NullString
	if imgURL != "" {
		imgURLNullable = sql.NullString{String: imgURL, Valid: true}
	} else {
		imgURLNullable = sql.NullString{Valid: false}
	}

	if err := insertBookWithAuthors(db, isbn, title, year, imgURLNullable, authorNames); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getAllBooks(db *sql.DB) ([]Book, error) {
	rows, err := db.Query("SELECT isbn, title, subtitle, year, imgURL FROM books")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var book Book
		if err := rows.Scan(&book.ISBN, &book.Title, &book.Subtitle, &book.Year, &book.ImgURL); err != nil {
			return nil, err
		}

		authors, err := getAuthorsForBook(db, book.ISBN)
		if err != nil {
			return nil, err
		}
		book.Authors = authors

		books = append(books, book)
	}

	return books, nil
}

func getAllIsbns(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT isbn FROM books")
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
	err := db.QueryRow("SELECT isbn, title, subtitle, year, imgURL FROM books WHERE isbn = ?", isbn).Scan(
		&book.ISBN, &book.Title, &book.Subtitle, &book.Year, &book.ImgURL)
	if err != nil {
		return Book{}, err
	}

	authors, err := getAuthorsForBook(db, book.ISBN)
	if err != nil {
		return Book{}, err
	}
	book.Authors = authors

	return book, nil
}

func getAuthorsForBook(db *sql.DB, isbn string) ([]Author, error) {
	rows, err := db.Query(`
           SELECT a.id, a.name 
    FROM authors a
    JOIN book_authors ba ON a.id = ba.author_id
    WHERE ba.book_isbn = ?`, isbn)
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

func insertBookWithAuthors(db *sql.DB, isbn, title string, year sql.NullInt64, imgURL sql.NullString, authorNames []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO books (isbn, title, year, imgURL) VALUES (?, ?, ?, ?)", isbn, title, year, imgURL)
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
