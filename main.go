package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
)


var db *sql.DB

func initDB() {

	var err error
	
	db, err = sql.Open("sqlite", "urls.db")

	if err != nil {
		log.Fatalf("Fatal error trying to load the database: %v", err)
	}

	createTableQuery := `CREATE TABLE IF NOT EXISTS urls (
		"short_code" TEXT PRIMARY KEY,
		"long_url" TEXT NOT NULL
	);`

	_, err = db.Exec(createTableQuery)

	if err != nil {
		log.Fatalf("Fatal error trying to create the table: %v", err)
	}

	fmt.Println("Database initialized successfully.")

}


// All characters that can be used to create the new URL
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
var sizeLetterBytes int = len(letterBytes)

func init() {
	
	// Run before the main function to make sure the generated URLs are different
	rand.Seed(time.Now().UnixNano())
}

func generateRandomString(n int) string {

	b := make([]byte, n)
	for i := range b {

		// Place a random character into the byte slice
		b[i] = letterBytes[rand.Intn(sizeLetterBytes)]

	}

	// Return the byte slice as a string
	return string(b)

}

func shortenHandler(w http.ResponseWriter, r *http.Request, amountOfCharactersToUse int) {

	if r.Method != http.MethodPost {

		http.Error(w, "Endpoint only accepts POST requests", http.StatusMethodNotAllowed)
		return

	}

	originalURL := r.FormValue("url")

	if originalURL == "" {

		http.Error(w, "The URL must not be empty", http.StatusBadRequest)
	
	}

	// Generate a random string
	randomString := generateRandomString(amountOfCharactersToUse)

	stmt, err := db.Prepare("INSERT INTO urls(short_code, long_url) VALUES(?, ?)")

	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error trying to prepare statement: %v", err)
		return
	}

	defer stmt.Close()

	_, err = stmt.Exec(randomString, originalURL)

	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error trying to execute statement: %v", err)
		return
	}

	shortURL := fmt.Sprintf("http://localhost:8080/%s", randomString)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Short URL: %s\n", shortURL)

}

func redirectHandler(w http.ResponseWriter, r *http.Request) {

	// Drop the leading "/"
	shortURL := r.URL.Path[1:]

	var originalURL string

	err := db.QueryRow("SELECT long_url FROM urls WHERE short_code = ?", shortURL).Scan(&originalURL)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "URL not found", http.StatusNotFound)
		} else {
			http.Error(w, "Server error", http.StatusInternalServerError)
			log.Printf("Erro within the database: %v", err)
		}
		return
	}

	if originalURL == "" {
		http.Error(w, "The URL must not be empty", http.StatusBadRequest)
	}

	http.Redirect(w, r, originalURL, http.StatusFound)

}

func main() {

	initDB()

	defer db.Close()

	fmt.Println("Type the number of characters in the shortened variable")
	amountOfCharactersToUse := 8
	fmt.Scan(&amountOfCharactersToUse)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/shorten" {
			shortenHandler(w, r, amountOfCharactersToUse)
		} else {
			// Any other path is treated as shortened URL
			redirectHandler(w, r)
		}
	})

	fmt.Println("Starting server on port 8080")

	err := http.ListenAndServe(":8080", nil)

	if err != nil {
		fmt.Printf("Initiation failed: %s\n", err)
	}
}