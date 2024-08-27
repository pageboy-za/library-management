-- create_tables.sql

-- Table for storing books
CREATE TABLE IF NOT EXISTS books (
    isbn TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    subtitle TEXT,
    year INTEGER
);

-- Table for storing authors
CREATE TABLE IF NOT EXISTS authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL
);

-- Join table to associate books with authors
CREATE TABLE IF NOT EXISTS book_authors (
    book_isbn TEXT NOT NULL,
    author_id INTEGER NOT NULL,
    FOREIGN KEY (book_isbn) REFERENCES books(isbn),
    FOREIGN KEY (author_id) REFERENCES authors(id),
    PRIMARY KEY (book_isbn, author_id)
);
