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
	"github.com/joho/godotenv"
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

	// Membaca file .env saat dijalankan secara lokal
	_ = godotenv.Load()

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
		log.Fatal("Gagal membuat koneksi database:", err)
	}

	defer db.Close()

	// Tes koneksi Neon
	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("Gagal terhubung ke Neon:", err)
	}

	log.Println("Database Neon berhasil terhubung")

	// =========================
	// ROUTES
	// =========================

	http.HandleFunc("/", homeHandler)

	http.HandleFunc("/penyanyi", penyanyiHandler)
	http.HandleFunc("/penyanyi/", penyanyiHandler)

	http.HandleFunc("/album", albumHandler)
	http.HandleFunc("/album/", albumHandler)

	http.HandleFunc("/lagu", laguHandler)
	http.HandleFunc("/lagu/", laguHandler)

	// Render memberikan PORT.
	// Lokal menggunakan 8080.
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	log.Println("Music API berjalan di port:", port)

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

	if r.URL.Path != "/" {
		errorResponse(w, http.StatusNotFound, "Endpoint tidak ditemukan")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Music API berhasil berjalan",
	})
}

// ============================================================
// PENYANYI
// ============================================================

func penyanyiHandler(w http.ResponseWriter, r *http.Request) {

	id, hasID := getIDFromPath(r.URL.Path)

	switch r.Method {

	case http.MethodGet:
		if hasID {
			getPenyanyiByID(w, r, id)
		} else {
			getPenyanyi(w, r)
		}

	case http.MethodPost:
		if hasID {
			errorResponse(w, 400, "POST tidak membutuhkan ID")
			return
		}
		createPenyanyi(w, r)

	case http.MethodPut:
		if !hasID {
			errorResponse(w, 400, "ID wajib diisi")
			return
		}
		updatePenyanyi(w, r, id)

	case http.MethodDelete:
		if !hasID {
			errorResponse(w, 400, "ID wajib diisi")
			return
		}
		deletePenyanyi(w, r, id)

	default:
		errorResponse(w, 405, "Method tidak diperbolehkan")
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

	data := []Penyanyi{}

	for rows.Next() {

		var p Penyanyi

		if err := rows.Scan(&p.ID, &p.Nama); err != nil {
			errorResponse(w, 500, err.Error())
			return
		}

		data = append(data, p)
	}

	jsonResponse(w, 200, data)
}

func getPenyanyiByID(w http.ResponseWriter, r *http.Request, id int) {

	var p Penyanyi

	err := db.QueryRow(
		r.Context(),
		"SELECT id, nama FROM penyanyi WHERE id = $1",
		id,
	).Scan(&p.ID, &p.Nama)

	if err == pgx.ErrNoRows {
		errorResponse(w, 404, "Penyanyi tidak ditemukan")
		return
	}

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, p)
}

func createPenyanyi(w http.ResponseWriter, r *http.Request) {

	var p Penyanyi

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		errorResponse(w, 400, "Format JSON tidak valid")
		return
	}

	if strings.TrimSpace(p.Nama) == "" {
		errorResponse(w, 400, "Nama penyanyi wajib diisi")
		return
	}

	err := db.QueryRow(
		r.Context(),
		`
		INSERT INTO penyanyi (nama)
		VALUES ($1)
		RETURNING id
		`,
		p.Nama,
	).Scan(&p.ID)

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	jsonResponse(w, 201, p)
}

func updatePenyanyi(w http.ResponseWriter, r *http.Request, id int) {

	var p Penyanyi

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		errorResponse(w, 400, "Format JSON tidak valid")
		return
	}

	if strings.TrimSpace(p.Nama) == "" {
		errorResponse(w, 400, "Nama penyanyi wajib diisi")
		return
	}

	err := db.QueryRow(
		r.Context(),
		`
		UPDATE penyanyi
		SET nama = $1
		WHERE id = $2
		RETURNING id, nama
		`,
		p.Nama,
		id,
	).Scan(&p.ID, &p.Nama)

	if err == pgx.ErrNoRows {
		errorResponse(w, 404, "Penyanyi tidak ditemukan")
		return
	}

	if err != nil {
		errorResponse(w, 500, err.Error())
		return
	}

	jsonResponse(w, 200, p)
}

func deletePenyanyi(w http.ResponseWriter, r *http.Request, id int) {

	command, err := db.Exec(
		r.Context(),
		"DELETE FROM penyanyi WHERE id = $1",
		id,
	)

	if err != nil {
		errorResponse(
			w,
			409,
			"Penyanyi tidak dapat dihapus karena masih digunakan oleh album",
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

// ============================================================
// ALBUM
// ============================================================

func albumHandler(w http.ResponseWriter, r *http.Request) {

	id, hasID := getIDFromPath(r.URL.Path)

	switch r.Method {

	case http.MethodGet:
		if hasID {
			getAlbumByID(w, r, id)
		} else {
			getAlbum(w, r)
		}

	case http.MethodPost:
		if hasID {
			errorResponse(w, 400, "POST tidak membutuhkan ID")
			return
		}
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

	data := []Album{}

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
		errorResponse(w, 400, "Format JSON tidak valid")
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
		errorResponse(w, 400, "Format JSON tidak valid")
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
			"Album tidak dapat dihapus karena masih digunakan oleh lagu",
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

// ============================================================
// LAGU
// ============================================================

func laguHandler(w http.ResponseWriter, r *http.Request) {

	id, hasID := getIDFromPath(r.URL.Path)

	switch r.Method {

	case http.MethodGet:
		if hasID {
			getLaguByID(w, r, id)
		} else {
			getLagu(w, r)
		}

	case http.MethodPost:
		if hasID {
			errorResponse(w, 400, "POST tidak membutuhkan ID")
			return
		}
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

	data := []Lagu{}

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
		errorResponse(w, 400, "Format JSON tidak valid")
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
		errorResponse(w, 400, "Format JSON tidak valid")
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

// ============================================================
// HELPER
// ============================================================

func getIDFromPath(path string) (int, bool) {

	parts := strings.Split(
		strings.Trim(path, "/"),
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