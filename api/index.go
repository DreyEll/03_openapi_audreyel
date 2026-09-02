package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

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
// SERVER
// =========================

type Server struct {
	DB *sql.DB
}

func NewServer(db *sql.DB) *Server {
	return &Server{
		DB: db,
	}
}

// =========================
// RESPONSE HELPER
// =========================

func writeJSON(
	w http.ResponseWriter,
	status int,
	data interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}

func errorJSON(
	w http.ResponseWriter,
	status int,
	message string,
) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

// =========================
// CORS
// =========================

func setCORS(w http.ResponseWriter) {
	w.Header().Set(
		"Access-Control-Allow-Origin",
		"*",
	)

	w.Header().Set(
		"Access-Control-Allow-Methods",
		"GET, POST, PUT, DELETE, OPTIONS",
	)

	w.Header().Set(
		"Access-Control-Allow-Headers",
		"Content-Type",
	)
}

// =========================
// MAIN HANDLER
// =========================

func (s *Server) Handler(
	w http.ResponseWriter,
	r *http.Request,
) {
	setCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.Trim(
		r.URL.Path,
		"/",
	)

	// =========================
	// HOME
	// =========================

	if path == "" {
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "Music API is running",
		})
		return
	}

	// =========================
	// PENYANYI
	// =========================

	if path == "penyanyi" {
		switch r.Method {

		case http.MethodGet:
			s.getPenyanyi(w, r)

		case http.MethodPost:
			s.createPenyanyi(w, r)

		default:
			methodNotAllowed(w)
		}

		return
	}

	if strings.HasPrefix(path, "penyanyi/") {

		id, err := getID(path, "penyanyi")

		if err != nil {
			errorJSON(
				w,
				http.StatusBadRequest,
				"ID tidak valid",
			)
			return
		}

		switch r.Method {

		case http.MethodGet:
			s.getPenyanyiByID(w, r, id)

		case http.MethodPut:
			s.updatePenyanyi(w, r, id)

		case http.MethodDelete:
			s.deletePenyanyi(w, r, id)

		default:
			methodNotAllowed(w)
		}

		return
	}

	// =========================
	// ALBUM
	// =========================

	if path == "album" {
		switch r.Method {

		case http.MethodGet:
			s.getAlbum(w, r)

		case http.MethodPost:
			s.createAlbum(w, r)

		default:
			methodNotAllowed(w)
		}

		return
	}

	if strings.HasPrefix(path, "album/") {

		id, err := getID(path, "album")

		if err != nil {
			errorJSON(
				w,
				http.StatusBadRequest,
				"ID tidak valid",
			)
			return
		}

		switch r.Method {

		case http.MethodGet:
			s.getAlbumByID(w, r, id)

		case http.MethodPut:
			s.updateAlbum(w, r, id)

		case http.MethodDelete:
			s.deleteAlbum(w, r, id)

		default:
			methodNotAllowed(w)
		}

		return
	}

	// =========================
	// LAGU
	// =========================

	if path == "lagu" {
		switch r.Method {

		case http.MethodGet:
			s.getLagu(w, r)

		case http.MethodPost:
			s.createLagu(w, r)

		default:
			methodNotAllowed(w)
		}

		return
	}

	if strings.HasPrefix(path, "lagu/") {

		id, err := getID(path, "lagu")

		if err != nil {
			errorJSON(
				w,
				http.StatusBadRequest,
				"ID tidak valid",
			)
			return
		}

		switch r.Method {

		case http.MethodGet:
			s.getLaguByID(w, r, id)

		case http.MethodPut:
			s.updateLagu(w, r, id)

		case http.MethodDelete:
			s.deleteLagu(w, r, id)

		default:
			methodNotAllowed(w)
		}

		return
	}

	errorJSON(
		w,
		http.StatusNotFound,
		"Endpoint tidak ditemukan",
	)
}

// =========================
// HELPER
// =========================

func methodNotAllowed(w http.ResponseWriter) {
	errorJSON(
		w,
		http.StatusMethodNotAllowed,
		"Method tidak diperbolehkan",
	)
}

func getID(
	path string,
	prefix string,
) (int, error) {

	idText := strings.TrimPrefix(
		path,
		prefix+"/",
	)

	return strconv.Atoi(idText)
}

// ========================================================
// PENYANYI
// ========================================================

func (s *Server) getPenyanyi(
	w http.ResponseWriter,
	r *http.Request,
) {
	rows, err := s.DB.QueryContext(
		r.Context(),
		`
		SELECT id, nama
		FROM penyanyi
		ORDER BY id
		`,
	)

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	defer rows.Close()

	data := []Penyanyi{}

	for rows.Next() {

		var p Penyanyi

		if err := rows.Scan(
			&p.ID,
			&p.Nama,
		); err != nil {

			errorJSON(
				w,
				http.StatusInternalServerError,
				err.Error(),
			)
			return
		}

		data = append(data, p)
	}

	writeJSON(w, http.StatusOK, data)
}

func (s *Server) getPenyanyiByID(
	w http.ResponseWriter,
	r *http.Request,
	id int,
) {
	var p Penyanyi

	err := s.DB.QueryRowContext(
		r.Context(),
		`
		SELECT id, nama
		FROM penyanyi
		WHERE id = $1
		`,
		id,
	).Scan(
		&p.ID,
		&p.Nama,
	)

	if err == sql.ErrNoRows {
		errorJSON(
			w,
			http.StatusNotFound,
			"Penyanyi tidak ditemukan",
		)
		return
	}

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (s *Server) createPenyanyi(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input struct {
		Nama string `json:"nama"`
	}

	if err := json.NewDecoder(
		r.Body,
	).Decode(&input); err != nil {

		errorJSON(
			w,
			http.StatusBadRequest,
			"Format JSON tidak valid",
		)
		return
	}

	input.Nama = strings.TrimSpace(input.Nama)

	if input.Nama == "" {
		errorJSON(
			w,
			http.StatusBadRequest,
			"Nama wajib diisi",
		)
		return
	}

	var p Penyanyi

	err := s.DB.QueryRowContext(
		r.Context(),
		`
		INSERT INTO penyanyi (nama)
		VALUES ($1)
		RETURNING id, nama
		`,
		input.Nama,
	).Scan(
		&p.ID,
		&p.Nama,
	)

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		p,
	)
}

func (s *Server) updatePenyanyi(
	w http.ResponseWriter,
	r *http.Request,
	id int,
) {
	var input struct {
		Nama string `json:"nama"`
	}

	if err := json.NewDecoder(
		r.Body,
	).Decode(&input); err != nil {

		errorJSON(
			w,
			http.StatusBadRequest,
			"Format JSON tidak valid",
		)
		return
	}

	input.Nama = strings.TrimSpace(input.Nama)

	if input.Nama == "" {
		errorJSON(
			w,
			http.StatusBadRequest,
			"Nama wajib diisi",
		)
		return
	}

	var p Penyanyi

	err := s.DB.QueryRowContext(
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
		&p.ID,
		&p.Nama,
	)

	if err == sql.ErrNoRows {
		errorJSON(
			w,
			http.StatusNotFound,
			"Penyanyi tidak ditemukan",
		)
		return
	}

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (s *Server) deletePenyanyi(
	w http.ResponseWriter,
	r *http.Request,
	id int,
) {
	var p Penyanyi

	err := s.DB.QueryRowContext(
		r.Context(),
		`
		DELETE FROM penyanyi
		WHERE id = $1
		RETURNING id, nama
		`,
		id,
	).Scan(
		&p.ID,
		&p.Nama,
	)

	if err == sql.ErrNoRows {
		errorJSON(
			w,
			http.StatusNotFound,
			"Penyanyi tidak ditemukan",
		)
		return
	}

	if err != nil {
		errorJSON(
			w,
			http.StatusConflict,
			"Penyanyi masih digunakan oleh album",
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		map[string]interface{}{
			"message": "Penyanyi berhasil dihapus",
			"data":    p,
		},
	)
}

// ========================================================
// ALBUM
// ========================================================

func (s *Server) getAlbum(
	w http.ResponseWriter,
	r *http.Request,
) {
	rows, err := s.DB.QueryContext(
		r.Context(),
		`
		SELECT id, nama_album, tahun, penyanyi_id
		FROM album
		ORDER BY id
		`,
	)

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
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

			errorJSON(
				w,
				http.StatusInternalServerError,
				err.Error(),
			)
			return
		}

		data = append(data, a)
	}

	writeJSON(w, http.StatusOK, data)
}

func (s *Server) getAlbumByID(
	w http.ResponseWriter,
	r *http.Request,
	id int,
) {
	var a Album

	err := s.DB.QueryRowContext(
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

	if err == sql.ErrNoRows {
		errorJSON(
			w,
			http.StatusNotFound,
			"Album tidak ditemukan",
		)
		return
	}

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(w, http.StatusOK, a)
}

func (s *Server) createAlbum(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input struct {
		NamaAlbum  string `json:"nama_album"`
		Tahun      int    `json:"tahun"`
		PenyanyiID int    `json:"penyanyi_id"`
	}

	if err := json.NewDecoder(
		r.Body,
	).Decode(&input); err != nil {

		errorJSON(
			w,
			http.StatusBadRequest,
			"Format JSON tidak valid",
		)
		return
	}

	if strings.TrimSpace(input.NamaAlbum) == "" {
		errorJSON(
			w,
			http.StatusBadRequest,
			"Nama album wajib diisi",
		)
		return
	}

	var exists bool

	err := s.DB.QueryRowContext(
		r.Context(),
		`
		SELECT EXISTS(
			SELECT 1
			FROM penyanyi
			WHERE id = $1
		)
		`,
		input.PenyanyiID,
	).Scan(&exists)

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	if !exists {
		errorJSON(
			w,
			http.StatusBadRequest,
			"Penyanyi tidak ditemukan",
		)
		return
	}

	var a Album

	err = s.DB.QueryRowContext(
		r.Context(),
		`
		INSERT INTO album
		(nama_album, tahun, penyanyi_id)
		VALUES ($1, $2, $3)
		RETURNING id, nama_album, tahun, penyanyi_id
		`,
		input.NamaAlbum,
		input.Tahun,
		input.PenyanyiID,
	).Scan(
		&a.ID,
		&a.NamaAlbum,
		&a.Tahun,
		&a.PenyanyiID,
	)

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		a,
	)
}

func (s *Server) updateAlbum(
	w http.ResponseWriter,
	r *http.Request,
	id int,
) {
	var input struct {
		NamaAlbum  string `json:"nama_album"`
		Tahun      int    `json:"tahun"`
		PenyanyiID int    `json:"penyanyi_id"`
	}

	if err := json.NewDecoder(
		r.Body,
	).Decode(&input); err != nil {

		errorJSON(
			w,
			http.StatusBadRequest,
			"Format JSON tidak valid",
		)
		return
	}

	var exists bool

	err := s.DB.QueryRowContext(
		r.Context(),
		`
		SELECT EXISTS(
			SELECT 1
			FROM penyanyi
			WHERE id = $1
		)
		`,
		input.PenyanyiID,
	).Scan(&exists)

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	if !exists {
		errorJSON(
			w,
			http.StatusBadRequest,
			"Penyanyi tidak ditemukan",
		)
		return
	}

	var a Album

	err = s.DB.QueryRowContext(
		r.Context(),
		`
		UPDATE album
		SET nama_album = $1,
		    tahun = $2,
		    penyanyi_id = $3
		WHERE id = $4
		RETURNING id, nama_album, tahun, penyanyi_id
		`,
		input.NamaAlbum,
		input.Tahun,
		input.PenyanyiID,
		id,
	).Scan(
		&a.ID,
		&a.NamaAlbum,
		&a.Tahun,
		&a.PenyanyiID,
	)

	if err == sql.ErrNoRows {
		errorJSON(
			w,
			http.StatusNotFound,
			"Album tidak ditemukan",
		)
		return
	}

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(w, http.StatusOK, a)
}

func (s *Server) deleteAlbum(
	w http.ResponseWriter,
	r *http.Request,
	id int,
) {
	var a Album

	err := s.DB.QueryRowContext(
		r.Context(),
		`
		DELETE FROM album
		WHERE id = $1
		RETURNING id, nama_album, tahun, penyanyi_id
		`,
		id,
	).Scan(
		&a.ID,
		&a.NamaAlbum,
		&a.Tahun,
		&a.PenyanyiID,
	)

	if err == sql.ErrNoRows {
		errorJSON(
			w,
			http.StatusNotFound,
			"Album tidak ditemukan",
		)
		return
	}

	if err != nil {
		errorJSON(
			w,
			http.StatusConflict,
			"Album masih digunakan oleh lagu",
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		map[string]interface{}{
			"message": "Album berhasil dihapus",
			"data":    a,
		},
	)
}

// ========================================================
// LAGU
// ========================================================

func (s *Server) getLagu(
	w http.ResponseWriter,
	r *http.Request,
) {
	rows, err := s.DB.QueryContext(
		r.Context(),
		`
		SELECT id, judul, album_id
		FROM lagu
		ORDER BY id
		`,
	)

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
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

			errorJSON(
				w,
				http.StatusInternalServerError,
				err.Error(),
			)
			return
		}

		data = append(data, l)
	}

	writeJSON(w, http.StatusOK, data)
}

func (s *Server) getLaguByID(
	w http.ResponseWriter,
	r *http.Request,
	id int,
) {
	var l Lagu

	err := s.DB.QueryRowContext(
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

	if err == sql.ErrNoRows {
		errorJSON(
			w,
			http.StatusNotFound,
			"Lagu tidak ditemukan",
		)
		return
	}

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(w, http.StatusOK, l)
}

func (s *Server) createLagu(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input struct {
		Judul   string `json:"judul"`
		AlbumID int    `json:"album_id"`
	}

	if err := json.NewDecoder(
		r.Body,
	).Decode(&input); err != nil {

		errorJSON(
			w,
			http.StatusBadRequest,
			"Format JSON tidak valid",
		)
		return
	}

	if strings.TrimSpace(input.Judul) == "" {
		errorJSON(
			w,
			http.StatusBadRequest,
			"Judul lagu wajib diisi",
		)
		return
	}

	var exists bool

	err := s.DB.QueryRowContext(
		r.Context(),
		`
		SELECT EXISTS(
			SELECT 1
			FROM album
			WHERE id = $1
		)
		`,
		input.AlbumID,
	).Scan(&exists)

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	if !exists {
		errorJSON(
			w,
			http.StatusBadRequest,
			"Album tidak ditemukan",
		)
		return
	}

	var l Lagu

	err = s.DB.QueryRowContext(
		r.Context(),
		`
		INSERT INTO lagu
		(judul, album_id)
		VALUES ($1, $2)
		RETURNING id, judul, album_id
		`,
		input.Judul,
		input.AlbumID,
	).Scan(
		&l.ID,
		&l.Judul,
		&l.AlbumID,
	)

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		l,
	)
}

func (s *Server) updateLagu(
	w http.ResponseWriter,
	r *http.Request,
	id int,
) {
	var input struct {
		Judul   string `json:"judul"`
		AlbumID int    `json:"album_id"`
	}

	if err := json.NewDecoder(
		r.Body,
	).Decode(&input); err != nil {

		errorJSON(
			w,
			http.StatusBadRequest,
			"Format JSON tidak valid",
		)
		return
	}

	if strings.TrimSpace(input.Judul) == "" {
		errorJSON(
			w,
			http.StatusBadRequest,
			"Judul lagu wajib diisi",
		)
		return
	}

	var exists bool

	err := s.DB.QueryRowContext(
		r.Context(),
		`
		SELECT EXISTS(
			SELECT 1
			FROM album
			WHERE id = $1
		)
		`,
		input.AlbumID,
	).Scan(&exists)

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	if !exists {
		errorJSON(
			w,
			http.StatusBadRequest,
			"Album tidak ditemukan",
		)
		return
	}

	var l Lagu

	err = s.DB.QueryRowContext(
		r.Context(),
		`
		UPDATE lagu
		SET judul = $1,
		    album_id = $2
		WHERE id = $3
		RETURNING id, judul, album_id
		`,
		input.Judul,
		input.AlbumID,
		id,
	).Scan(
		&l.ID,
		&l.Judul,
		&l.AlbumID,
	)

	if err == sql.ErrNoRows {
		errorJSON(
			w,
			http.StatusNotFound,
			"Lagu tidak ditemukan",
		)
		return
	}

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(w, http.StatusOK, l)
}

func (s *Server) deleteLagu(
	w http.ResponseWriter,
	r *http.Request,
	id int,
) {
	var l Lagu

	err := s.DB.QueryRowContext(
		r.Context(),
		`
		DELETE FROM lagu
		WHERE id = $1
		RETURNING id, judul, album_id
		`,
		id,
	).Scan(
		&l.ID,
		&l.Judul,
		&l.AlbumID,
	)

	if err == sql.ErrNoRows {
		errorJSON(
			w,
			http.StatusNotFound,
			"Lagu tidak ditemukan",
		)
		return
	}

	if err != nil {
		errorJSON(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		map[string]interface{}{
			"message": "Lagu berhasil dihapus",
			"data":    l,
		},
	)
}