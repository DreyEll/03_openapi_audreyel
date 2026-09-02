package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"music-api/api"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {

	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		log.Fatal("DATABASE_URL belum diatur")
	}

	db, err := sql.Open(
		"pgx",
		databaseURL,
	)

	if err != nil {
		log.Fatal(
			"Gagal membuka database:",
			err,
		)
	}

	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(
			"Gagal terhubung ke Neon:",
			err,
		)
	}

	server := api.NewServer(db)

	http.HandleFunc(
		"/",
		server.Handler,
	)

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	log.Println(
		"=================================",
	)

	log.Println(
		"Music API berhasil dijalankan",
	)

	log.Println(
		"http://localhost:" + port,
	)

	log.Println(
		"=================================",
	)

	err = http.ListenAndServe(
		":"+port,
		nil,
	)

	if err != nil {
		log.Fatal(err)
	}
}