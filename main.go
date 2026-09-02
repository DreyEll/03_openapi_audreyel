package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

// =========================
// MODEL
// =========================

type Penyanyi struct {
	ID   int    `json:"id"`
	Nama string `json:"nama"`
}

type Album struct {
	ID         int    `json:"id"`
	NamaAlbum  string `json:"nama_album"`
	Tahun      int    `json:"tahun"`
	PenyanyiID int    `json:"penyanyi_id"`
}

type Lagu struct {
	ID      int    `json:"id"`
	Judul   string `json:"judul"`
	AlbumID int    `json:"album_id"`
}

// =========================
// RESPONSE
// =========================

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]string{
		"error": message,
	})
}

// =========================
// MAIN
// =========================

func main() {

	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		log.Fatal("DATABASE_URL belum diatur")
	}

	var err error

	db, err = pgxpool.New(
		context.Background(),
		databaseURL,
	)

	if err != nil {
		log.Fatal("Gagal koneksi database:", err)
	}

	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("Neon tidak bisa diakses:", err)
	}

	log.Println("Database Neon berhasil terhubung")

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/penyanyi", penyanyiHandler)
	http.HandleFunc("/album", albumHandler)
	http.HandleFunc("/lagu", laguHandler)

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	log.Println("Music API berjalan di port", port)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

// =========================
// HOME
// =========================

func homeHandler(w http.ResponseWriter, r *http.Request) {

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Music API berhasil berjalan",
	})
}

// ======================================================
// PENYANYI
// ======================================================

func penyanyiHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodGet:
		getPenyanyi(w, r)

	case http.MethodPost:
		createPenyanyi(w, r)

	case http.MethodPut:
		updatePenyanyi(w, r)

	case http.MethodDelete:
		deletePenyanyi(w, r)

	default:
		errorResponse(
			w,
			http.StatusMethodNotAllowed,
			"Method tidak diperbolehkan",
		)
	}
}

func getPenyanyi(w http.ResponseWriter, r *http.Request) {

	rows, err := db.Query(
		r.Context(),
		"SELECT id, nama FROM penyanyi ORDER BY id",
	)

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	defer rows.Close()

	var data []Penyanyi

	for rows.Next() {

		var p Penyanyi

		err := rows.Scan(
			&p.ID,
			&p.Nama,
		)

		if err != nil {
			errorResponse(w, 500, err.Error())
			return
		}

		data = append(data, p)
	}

	jsonResponse(w, 200, data)
}

func createPenyanyi(w http.ResponseWriter, r *http.Request) {

	var input Penyanyi

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errorResponse(w, 400, "JSON tidak valid")
		return
	}

	if strings.TrimSpace(input.Nama) == "" {
		errorResponse(w, 400, "Nama wajib diisi")
		return
	}

	err := db.QueryRow(
		r.Context(),
		"INSERT INTO penyanyi (nama) VALUES ($1) RETURNING id",
		input.Nama,
	).Scan(&input.ID)

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	jsonResponse(w, 201, input)
}

func updatePenyanyi(w http.ResponseWriter, r *http.Request) {

	id, err := getID(r)

	if err != nil {
		errorResponse(w, 400, "ID tidak valid")
		return
	}

	var input Penyanyi

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errorResponse(w, 400, "JSON tidak valid")
		return
	}

	err = db.QueryRow(
		r.Context(),
		`
		UPDATE penyanyi
		SET nama = $1
		WHERE id = $2
		RETURNING id, nama
		`,
		input.Nama,
		id,
	).Scan(
		&input.ID,
		&input.Nama,
	)

	if err == pgx.ErrNoRows {
		errorResponse(w, 404, "Penyanyi tidak ditemukan")
		return
	}

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, input)
}

func deletePenyanyi(w http.ResponseWriter, r *http.Request) {

	id, err := getID(r)

	if err != nil {
		errorResponse(w, 400, "ID tidak valid")
		return
	}

	command, err := db.Exec(
		r.Context(),
		"DELETE FROM penyanyi WHERE id = $1",
		id,
	)

	if err != nil {
		errorResponse(
			w,
			409,
			"Penyanyi masih digunakan oleh album",
		)
		return
	}

	if command.RowsAffected() == 0 {
		errorResponse(w, 404, "Penyanyi tidak ditemukan")
		return
	}

	jsonResponse(w, 200, map[string]string{
		"message": "Penyanyi berhasil dihapus",
	})
}

// ======================================================
// ALBUM
// ======================================================

func albumHandler(w http.ResponseWriter, r *http.Request) {

	id, hasID := getOptionalID(r)

	switch r.Method {

	case http.MethodGet:

		if hasID {
			getAlbumByID(w, r, id)
		} else {
			getAlbum(w, r)
		}

	case http.MethodPost:
		createAlbum(w, r)

	case http.MethodPut:

		if !hasID {
			errorResponse(w, 400, "ID wajib diisi")
			return
		}

		updateAlbum(w, r, id)

	case http.MethodDelete:

		if !hasID {
			errorResponse(w, 400, "ID wajib diisi")
			return
		}

		deleteAlbum(w, r, id)

	default:
		errorResponse(w, 405, "Method tidak diperbolehkan")
	}
}

func getAlbum(w http.ResponseWriter, r *http.Request) {

	rows, err := db.Query(
		r.Context(),
		`
		SELECT id, nama_album, tahun, penyanyi_id
		FROM album
		ORDER BY id
		`,
	)

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	defer rows.Close()

	var data []Album

	for rows.Next() {

		var a Album

		if err := rows.Scan(
			&a.ID,
			&a.NamaAlbum,
			&a.Tahun,
			&a.PenyanyiID,
		); err != nil {
			errorResponse(w, 500, err.Error())
			return
		}

		data = append(data, a)
	}

	jsonResponse(w, 200, data)
}

func getAlbumByID(w http.ResponseWriter, r *http.Request, id int) {

	var a Album

	err := db.QueryRow(
		r.Context(),
		`
		SELECT id, nama_album, tahun, penyanyi_id
		FROM album
		WHERE id = $1
		`,
		id,
	).Scan(
		&a.ID,
		&a.NamaAlbum,
		&a.Tahun,
		&a.PenyanyiID,
	)

	if err == pgx.ErrNoRows {
		errorResponse(w, 404, "Album tidak ditemukan")
		return
	}

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, a)
}

func createAlbum(w http.ResponseWriter, r *http.Request) {

	var a Album

	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		errorResponse(w, 400, "JSON tidak valid")
		return
	}

	err := db.QueryRow(
		r.Context(),
		`
		INSERT INTO album
		(nama_album, tahun, penyanyi_id)
		VALUES ($1, $2, $3)
		RETURNING id
		`,
		a.NamaAlbum,
		a.Tahun,
		a.PenyanyiID,
	).Scan(&a.ID)

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	jsonResponse(w, 201, a)
}

func updateAlbum(w http.ResponseWriter, r *http.Request, id int) {

	var a Album

	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		errorResponse(w, 400, "JSON tidak valid")
		return
	}

	err := db.QueryRow(
		r.Context(),
		`
		UPDATE album
		SET nama_album = $1,
		    tahun = $2,
		    penyanyi_id = $3
		WHERE id = $4
		RETURNING id, nama_album, tahun, penyanyi_id
		`,
		a.NamaAlbum,
		a.Tahun,
		a.PenyanyiID,
		id,
	).Scan(
		&a.ID,
		&a.NamaAlbum,
		&a.Tahun,
		&a.PenyanyiID,
	)

	if err == pgx.ErrNoRows {
		errorResponse(w, 404, "Album tidak ditemukan")
		return
	}

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, a)
}

func deleteAlbum(w http.ResponseWriter, r *http.Request, id int) {

	command, err := db.Exec(
		r.Context(),
		"DELETE FROM album WHERE id = $1",
		id,
	)

	if err != nil {
		errorResponse(
			w,
			409,
			"Album masih digunakan oleh lagu",
		)
		return
	}

	if command.RowsAffected() == 0 {
		errorResponse(w, 404, "Album tidak ditemukan")
		return
	}

	jsonResponse(w, 200, map[string]string{
		"message": "Album berhasil dihapus",
	})
}

// ======================================================
// LAGU
// ======================================================

func laguHandler(w http.ResponseWriter, r *http.Request) {

	id, hasID := getOptionalID(r)

	switch r.Method {

	case http.MethodGet:

		if hasID {
			getLaguByID(w, r, id)
		} else {
			getLagu(w, r)
		}

	case http.MethodPost:
		createLagu(w, r)

	case http.MethodPut:

		if !hasID {
			errorResponse(w, 400, "ID wajib diisi")
			return
		}

		updateLagu(w, r, id)

	case http.MethodDelete:

		if !hasID {
			errorResponse(w, 400, "ID wajib diisi")
			return
		}

		deleteLagu(w, r, id)

	default:
		errorResponse(w, 405, "Method tidak diperbolehkan")
	}
}

func getLagu(w http.ResponseWriter, r *http.Request) {

	rows, err := db.Query(
		r.Context(),
		`
		SELECT id, judul, album_id
		FROM lagu
		ORDER BY id
		`,
	)

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	defer rows.Close()

	var data []Lagu

	for rows.Next() {

		var l Lagu

		if err := rows.Scan(
			&l.ID,
			&l.Judul,
			&l.AlbumID,
		); err != nil {
			errorResponse(w, 500, err.Error())
			return
		}

		data = append(data, l)
	}

	jsonResponse(w, 200, data)
}

func getLaguByID(w http.ResponseWriter, r *http.Request, id int) {

	var l Lagu

	err := db.QueryRow(
		r.Context(),
		`
		SELECT id, judul, album_id
		FROM lagu
		WHERE id = $1
		`,
		id,
	).Scan(
		&l.ID,
		&l.Judul,
		&l.AlbumID,
	)

	if err == pgx.ErrNoRows {
		errorResponse(w, 404, "Lagu tidak ditemukan")
		return
	}

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, l)
}

func createLagu(w http.ResponseWriter, r *http.Request) {

	var l Lagu

	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		errorResponse(w, 400, "JSON tidak valid")
		return
	}

	err := db.QueryRow(
		r.Context(),
		`
		INSERT INTO lagu
		(judul, album_id)
		VALUES ($1, $2)
		RETURNING id
		`,
		l.Judul,
		l.AlbumID,
	).Scan(&l.ID)

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	jsonResponse(w, 201, l)
}

func updateLagu(w http.ResponseWriter, r *http.Request, id int) {

	var l Lagu

	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		errorResponse(w, 400, "JSON tidak valid")
		return
	}

	err := db.QueryRow(
		r.Context(),
		`
		UPDATE lagu
		SET judul = $1,
		    album_id = $2
		WHERE id = $3
		RETURNING id, judul, album_id
		`,
		l.Judul,
		l.AlbumID,
		id,
	).Scan(
		&l.ID,
		&l.Judul,
		&l.AlbumID,
	)

	if err == pgx.ErrNoRows {
		errorResponse(w, 404, "Lagu tidak ditemukan")
		return
	}

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, l)
}

func deleteLagu(w http.ResponseWriter, r *http.Request, id int) {

	command, err := db.Exec(
		r.Context(),
		"DELETE FROM lagu WHERE id = $1",
		id,
	)

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	if command.RowsAffected() == 0 {
		errorResponse(w, 404, "Lagu tidak ditemukan")
		return
	}

	jsonResponse(w, 200, map[string]string{
		"message": "Lagu berhasil dihapus",
	})
}

// =========================
// ID HELPER
// =========================

func getID(r *http.Request) (int, error) {

	parts := strings.Split(
		strings.Trim(r.URL.Path, "/"),
		"/",
	)

	return strconv.Atoi(parts[len(parts)-1])
}

func getOptionalID(r *http.Request) (int, bool) {

	parts := strings.Split(
		strings.Trim(r.URL.Path, "/"),
		"/",
	)

	if len(parts) < 2 {
		return 0, false
	}

	id, err := strconv.Atoi(parts[len(parts)-1])

	if err != nil {
		return 0, false
	}

	return id, true
}