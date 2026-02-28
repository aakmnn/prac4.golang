package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Movie struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

func mustEnv(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		log.Fatalf("missing env var: %s", key)
	}
	return v
}

func openDB() *sql.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		mustEnv("DB_HOST"),
		mustEnv("DB_PORT"),
		mustEnv("DB_USER"),
		mustEnv("DB_PASSWORD"),
		mustEnv("DB_NAME"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	return db
}

func waitForDB(db *sql.DB) {
	for {
		if err := db.Ping(); err == nil {
			return
		}
		log.Println("Waiting for database...")
		time.Sleep(1 * time.Second)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func main() {
	port := os.Getenv("PORT")
	if strings.TrimSpace(port) == "" {
		port = "8080"
	}

	db := openDB()
	defer db.Close()

	waitForDB(db)
	log.Println("Database connected")
	log.Println("Starting the Server...")

	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Collection endpoints
	mux.HandleFunc("/movies", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rows, err := db.Query(`SELECT id, title FROM movies ORDER BY id`)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			defer rows.Close()

			var out []Movie
			for rows.Next() {
				var m Movie
				if err := rows.Scan(&m.ID, &m.Title); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				out = append(out, m)
			}
			writeJSON(w, http.StatusOK, out)

		case http.MethodPost:
			var in struct {
				Title string `json:"title"`
			}
			if err := readJSON(r, &in); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
				return
			}
			in.Title = strings.TrimSpace(in.Title)
			if in.Title == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
				return
			}

			var id int64
			err := db.QueryRow(`INSERT INTO movies (title) VALUES ($1) RETURNING id`, in.Title).Scan(&id)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusCreated, Movie{ID: id, Title: in.Title})

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Item endpoints: /movies/{id}
	mux.HandleFunc("/movies/", func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/movies/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}

		switch r.Method {
		case http.MethodGet:
			var m Movie
			err := db.QueryRow(`SELECT id, title FROM movies WHERE id=$1`, id).Scan(&m.ID, &m.Title)
			if err == sql.ErrNoRows {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, m)

		case http.MethodPut:
			var in struct {
				Title string `json:"title"`
			}
			if err := readJSON(r, &in); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
				return
			}
			in.Title = strings.TrimSpace(in.Title)
			if in.Title == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
				return
			}

			res, err := db.Exec(`UPDATE movies SET title=$1 WHERE id=$2`, in.Title, id)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			aff, _ := res.RowsAffected()
			if aff == 0 {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}
			writeJSON(w, http.StatusOK, Movie{ID: id, Title: in.Title})

		case http.MethodDelete:
			res, err := db.Exec(`DELETE FROM movies WHERE id=$1`, id)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			aff, _ := res.RowsAffected()
			if aff == 0 {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
