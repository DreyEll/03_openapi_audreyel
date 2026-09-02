package api

import (
	"database/sql"
	"net/http"
	"os"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	dbInstance *sql.DB
	dbOnce     sync.Once
	dbError    error
)

func Handler(w http.ResponseWriter, r *http.Request) {

	dbOnce.Do(func() {

		databaseURL := os.Getenv("DATABASE_URL")

		if databaseURL == "" {
			dbError = &databaseError{
				message: "DATABASE_URL belum diatur",
			}
			return
		}

		dbInstance, dbError = sql.Open(
			"pgx",
			databaseURL,
		)

		if dbError != nil {
			return
		}

		dbError = dbInstance.Ping()
	})

	if dbError != nil {

		http.Error(
			w,
			dbError.Error(),
			http.StatusInternalServerError,
		)

		return
	}

	server := NewServer(dbInstance)

	server.Handler(w, r)
}

type databaseError struct {
	message string
}

func (e *databaseError) Error() string {
	return e.message
}