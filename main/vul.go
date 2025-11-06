// test_vulnerabilities.go - DELIBERATE VULNERABILITIES FOR TESTING
package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
)

// SQL Injection - Should trigger go/sql-injection
func unsafeQuery(db *sql.DB, userInput string) {
	query := "SELECT * FROM users WHERE name = '" + userInput + "'"
	db.Query(query) // VULNERABLE
}

// Command Injection - Should trigger go/command-injection
func unsafeCommand(userInput string) {
	cmd := exec.Command("sh", "-c", "echo "+userInput)
	cmd.Run() // VULNERABLE
}

// Path Traversal - Should trigger go/path-injection
func unsafeFileRead(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	data, _ := os.ReadFile(filename) // VULNERABLE
	w.Write(data)
}

// Hardcoded Credentials - Should trigger go/hardcoded-credentials
func connectDB() {
	password := "admin123" // VULNERABLE
	fmt.Println(password)

}
// Insecure Random - Should trigger go/insecure-randomness
func weakRandom() {
	// This won't trigger without math/rand usage, but adding it:
	_ = rand.Int()
}
