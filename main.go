package main

import (
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Initialize the database
	db, err := InitializeDatabase("./library.db")
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Set up the routes
	http.HandleFunc("/books", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getAllBooksHandler(w, r, db)
		} else {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/books/", func(w http.ResponseWriter, r *http.Request) {
		isbn := r.URL.Path[len("/books/"):]
		if isbn == "" {
			http.Error(w, "ISBN required", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			getBookHandler(w, r, db, isbn)
		default:
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/submit-book", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			submitBookHandler(w, r, db)
		} else {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/isbns", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getAllIsbnsHandler(w, r, db)
		} else {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	})

	// Start the server
	log.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
