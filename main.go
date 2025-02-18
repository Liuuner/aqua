package main

import (
	"database/sql"
	"embed"
	"fmt"
	"github.com/google/uuid"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// --- Global Variables ---
var (
	db        *sql.DB
	templates *template.Template
)

var jwtKey = []byte(os.Getenv("JWT_KEY"))

// Claims defines the structure for our JWT payload.
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

//go:embed templates/*
var templateFS embed.FS

func main() {
	/*if len(jwtKey) == 0 {
		// generate random key
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			log.Fatal(err)
		}
		jwtKey = key
	}*/
	port := "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	dataSourceName := "aqua.db"
	if os.Getenv("DATABASE_URL") != "" {
		dataSourceName = os.Getenv("DATABASE_URL")
	}

	var err error
	db, err = sql.Open("sqlite", dataSourceName)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize database tables.
	if err := initDB(); err != nil {
		log.Fatal(err)
	}

	templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

	// Setup routes.
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/signup", signupHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/increment", incrementHandler)
	http.HandleFunc("/history", historyHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	fmt.Println("Server started on http://localhost:" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initDB() error {
	// Create the "users" table.
	usersTable := `
 CREATE TABLE IF NOT EXISTS users (
  id STRING PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL
 );`
	if _, err := db.Exec(usersTable); err != nil {
		return err
	}

	// Create the "water_counts" table.
	waterCountsTable := `
 CREATE TABLE IF NOT EXISTS water_counts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id STRING NOT NULL,
  date TEXT NOT NULL,
  count_330ml INTEGER NOT NULL DEFAULT 0,
  count_500ml INTEGER NOT NULL DEFAULT 0,
  count_750ml INTEGER NOT NULL DEFAULT 0,
  count_1000ml INTEGER NOT NULL DEFAULT 0,
  count_1500ml INTEGER NOT NULL DEFAULT 0,
  UNIQUE(user_id, date),
  FOREIGN KEY(user_id) REFERENCES users(id)
 );`
	_, err := db.Exec(waterCountsTable)
	return err
}

// --- JWT Session Management ---

// generateJWT creates a JWT for the given user id with a 24-hour expiration.
func generateJWT(userID string) (string, error) {
	expirationTime := time.Now().Add(24 * 7 * time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// getUserID retrieves the user ID from the JWT stored in the cookie.
func getUserID(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("token")
	if err != nil {
		return "", false
	}
	tokenStr := cookie.Value
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		return "", false
	}
	return claims.UserID, true
}

// setJWT creates a JWT token and sets it in an HttpOnly cookie.
func setJWT(w http.ResponseWriter, userID string) error {
	tokenString, err := generateJWT(userID)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // enable when using HTTPS
		Expires:  time.Now().Add(24 * time.Hour),
	})
	return nil
}

// clearJWT clears the authentication token cookie.
func clearJWT(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
	})
}

// --- HTTP Handlers ---

// homeHandler shows the counter page if the user is logged in.
func homeHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserID(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	today := time.Now().Format("02-01-2006")
	var counts struct {
		Count330ml  int
		Count500ml  int
		Count750ml  int
		Count1000ml int
		Count1500ml int
		Total       float64
		Increments  []string
	}
	err := db.QueryRow("SELECT count_330ml, count_500ml, count_750ml, count_1000ml, count_1500ml FROM water_counts WHERE user_id = ? AND date = ?", userID, today).Scan(&counts.Count330ml, &counts.Count500ml, &counts.Count750ml, &counts.Count1000ml, &counts.Count1500ml)
	if err != nil {
		if err == sql.ErrNoRows {
			counts = struct {
				Count330ml  int
				Count500ml  int
				Count750ml  int
				Count1000ml int
				Count1500ml int
				Total       float64
				Increments  []string
			}{Increments: []string{"330ml", "500ml", "750ml", "1000ml", "1500ml"}}
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	} else {
		counts.Total = float64(counts.Count330ml*330+counts.Count500ml*500+counts.Count750ml*750+counts.Count1000ml*1000+counts.Count1500ml*1500) / 1000.0
		counts.Increments = []string{"330ml", "500ml", "750ml", "1000ml", "1500ml"}
	}

	if err := templates.ExecuteTemplate(w, "counter.html", counts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// loginHandler handles both GET (display form) and POST (process login).
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if err := templates.ExecuteTemplate(w, "login.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var userID string
	var passwordHash string
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", username).Scan(&userID, &passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := setJWT(w, userID); err != nil {
		http.Error(w, "Could not create token", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// signupHandler handles user registration.
func signupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if err := templates.ExecuteTemplate(w, "signup.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	id := uuid.NewString()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)", id, username, string(hash))
	if err != nil {
		http.Error(w, "Error creating user (username may already be taken)", http.StatusInternalServerError)
		return
	}

	if err := setJWT(w, id); err != nil {
		http.Error(w, "Could not create token", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// logoutHandler logs out the user.
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	clearJWT(w, r)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// incrementHandler is called via HTMX to update the counter.
func incrementHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, loggedIn := getUserID(r)
	if !loggedIn {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	increment := r.FormValue("increment")
	var column string
	switch increment {
	case "330ml":
		column = "count_330ml"
	case "500ml":
		column = "count_500ml"
	case "750ml":
		column = "count_750ml"
	case "1000ml":
		column = "count_1000ml"
	case "1500ml":
		column = "count_1500ml"
	default:
		http.Error(w, "Invalid increment", http.StatusBadRequest)
		return
	}

	today := time.Now().Format("02-01-2006")
	// Update today's counter; if no record exists, insert one.
	res, err := db.Exec(fmt.Sprintf("UPDATE water_counts SET %s = %s + 1 WHERE user_id = ? AND date = ?", column, column), userID, today)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		_, err = db.Exec(fmt.Sprintf("INSERT INTO water_counts (user_id, date, %s) VALUES (?, ?, 1)", column), userID, today)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	var counts struct {
		Count330ml  int
		Count500ml  int
		Count750ml  int
		Count1000ml int
		Count1500ml int
		Total       float64
	}
	err = db.QueryRow("SELECT count_330ml, count_500ml, count_750ml, count_1000ml, count_1500ml FROM water_counts WHERE user_id = ? AND date = ?", userID, today).Scan(&counts.Count330ml, &counts.Count500ml, &counts.Count750ml, &counts.Count1000ml, &counts.Count1500ml)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	counts.Total = float64(counts.Count330ml*330+counts.Count500ml*500+counts.Count750ml*750+counts.Count1000ml*1000+counts.Count1500ml*1500) / 1000.0

	// Return the updated counts and total
	if err := templates.ExecuteTemplate(w, "counter_table.html", counts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// historyHandler shows the history of the total amount per day.
func historyHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserID(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	rows, err := db.Query("SELECT date, (count_330ml * 330 + count_500ml * 500 + count_750ml * 750 + count_1000ml * 1000 + count_1500ml * 1500) / 1000.0 as total FROM water_counts WHERE user_id = ? ORDER BY date DESC", userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []struct {
		Date  string
		Total float64
	}

	for rows.Next() {
		var record struct {
			Date  string
			Total float64
		}
		if err := rows.Scan(&record.Date, &record.Total); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		history = append(history, record)
	}

	if err := templates.ExecuteTemplate(w, "history.html", history); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
