package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/graphql-go/graphql"
	"github.com/rs/cors"
)

// ============================================================
// COUNTER N+1
// ============================================================

// Menghitung berapa kali Query.penyanyi dipanggil
var queryPenyanyiCallCount = 0

// Menghitung berapa kali resolver relasi Penyanyi.album dipanggil
var resolverCallCount = 0

// ============================================================
// CREATE GRAPHQL SCHEMA
// ============================================================

func createGraphQLSchema() (graphql.Schema, error) {

	// ========================================================
	// ALBUM TYPE
	// ========================================================

	albumType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Album",

		Fields: graphql.Fields{
			"nama_album": &graphql.Field{
				Type: graphql.String,
			},

			"tahun": &graphql.Field{
				Type: graphql.Int,
			},
		},
	})

	// ========================================================
	// LAGU TYPE
	// ========================================================

	laguType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Lagu",

		Fields: graphql.Fields{

			"id": &graphql.Field{
				Type: graphql.Int,
			},

			"judul": &graphql.Field{
				Type: graphql.String,
			},

			"album_id": &graphql.Field{
				Type: graphql.Int,
			},
		},
	})

	// ========================================================
	// PENYANYI TYPE
	// ========================================================

	penyanyiType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Penyanyi",

		Fields: graphql.Fields{

			"id": &graphql.Field{
				Type: graphql.Int,
			},

			"nama": &graphql.Field{
				Type: graphql.String,
			},

			// ------------------------------------------------
			// Relasi Penyanyi -> Album
			// ------------------------------------------------

			"album": &graphql.Field{
				Type: graphql.NewList(albumType),

				Resolve: func(p graphql.ResolveParams) (interface{}, error) {

					// Counter resolver relasi
					resolverCallCount++

					log.Printf(
						"Penyanyi.album resolver dipanggil ke-%d",
						resolverCallCount,
					)

					penyanyi, ok := p.Source.(map[string]interface{})
					if !ok {
						return nil, nil
					}

					penyanyiID, ok := penyanyi["id"].(int)
					if !ok {
						return nil, nil
					}

					rows, err := db.Query(
						context.Background(),
						`
						SELECT nama_album, tahun
						FROM album
						WHERE penyanyi_id = $1
						ORDER BY id
						`,
						penyanyiID,
					)

					if err != nil {
						return nil, err
					}

					defer rows.Close()

					albumList := []interface{}{}

					for rows.Next() {

						var namaAlbum string
						var tahun int

						err := rows.Scan(
							&namaAlbum,
							&tahun,
						)

						if err != nil {
							return nil, err
						}

						albumList = append(
							albumList,
							map[string]interface{}{
								"nama_album": namaAlbum,
								"tahun":      tahun,
							},
						)
					}

					if err := rows.Err(); err != nil {
						return nil, err
					}

					return albumList, nil
				},
			},
		},
	})

	// ========================================================
	// QUERY TYPE
	// ========================================================

	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",

		Fields: graphql.Fields{

			// ==================================================
			// QUERY PENYANYI
			// ==================================================

			"penyanyi": &graphql.Field{
				Type: graphql.NewList(penyanyiType),

				Resolve: func(p graphql.ResolveParams) (interface{}, error) {

					queryPenyanyiCallCount++

					log.Printf(
						"Query.penyanyi dipanggil ke-%d",
						queryPenyanyiCallCount,
					)

					rows, err := db.Query(
						context.Background(),
						`
						SELECT id, nama
						FROM penyanyi
						ORDER BY id
						`,
					)

					if err != nil {
						return nil, err
					}

					defer rows.Close()

					penyanyiList := []interface{}{}

					for rows.Next() {

						var id int
						var nama string

						err := rows.Scan(
							&id,
							&nama,
						)

						if err != nil {
							return nil, err
						}

						penyanyiList = append(
							penyanyiList,
							map[string]interface{}{
								"id":   id,
								"nama": nama,
							},
						)
					}

					if err := rows.Err(); err != nil {
						return nil, err
					}

					return penyanyiList, nil
				},
			},

			// ==================================================
			// QUERY LAGU
			// ==================================================

			"lagu": &graphql.Field{
				Type: graphql.NewList(laguType),

				Resolve: func(p graphql.ResolveParams) (interface{}, error) {

					rows, err := db.Query(
						context.Background(),
						`
						SELECT id, judul, album_id
						FROM lagu
						ORDER BY id
						`,
					)

					if err != nil {
						return nil, err
					}

					defer rows.Close()

					laguList := []interface{}{}

					for rows.Next() {

						var id int
						var judul string
						var albumID int

						err := rows.Scan(
							&id,
							&judul,
							&albumID,
						)

						if err != nil {
							return nil, err
						}

						laguList = append(
							laguList,
							map[string]interface{}{
								"id":       id,
								"judul":    judul,
								"album_id": albumID,
							},
						)
					}

					if err := rows.Err(); err != nil {
						return nil, err
					}

					return laguList, nil
				},
			},
		},
	})

	// ========================================================
	// MUTATION TYPE
	// ========================================================

	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",

		Fields: graphql.Fields{

			// ==================================================
			// CREATE LAGU
			// ==================================================

			"createLagu": &graphql.Field{

				Type: laguType,

				Args: graphql.FieldConfigArgument{

					"judul": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},

					"album_id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},

				Resolve: func(p graphql.ResolveParams) (interface{}, error) {

					judul := p.Args["judul"].(string)
					albumID := p.Args["album_id"].(int)

					var id int

					err := db.QueryRow(
						context.Background(),
						`
						INSERT INTO lagu (judul, album_id)
						VALUES ($1, $2)
						RETURNING id
						`,
						judul,
						albumID,
					).Scan(&id)

					if err != nil {
						return nil, err
					}

					return map[string]interface{}{
						"id":       id,
						"judul":    judul,
						"album_id": albumID,
					}, nil
				},
			},

			// ==================================================
			// UPDATE LAGU
			// ==================================================

			"updateLagu": &graphql.Field{

				Type: laguType,

				Args: graphql.FieldConfigArgument{

					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},

					"judul": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},

				Resolve: func(p graphql.ResolveParams) (interface{}, error) {

					id := p.Args["id"].(int)
					judul := p.Args["judul"].(string)

					result, err := db.Exec(
						context.Background(),
						`
						UPDATE lagu
						SET judul = $1
						WHERE id = $2
						`,
						judul,
						id,
					)

					if err != nil {
						return nil, err
					}

					if result.RowsAffected() == 0 {
						return nil, nil
					}

					var albumID int

					err = db.QueryRow(
						context.Background(),
						`
						SELECT album_id
						FROM lagu
						WHERE id = $1
						`,
						id,
					).Scan(&albumID)

					if err != nil {
						return nil, err
					}

					return map[string]interface{}{
						"id":       id,
						"judul":    judul,
						"album_id": albumID,
					}, nil
				},
			},

			// ==================================================
			// DELETE LAGU
			// ==================================================

			"deleteLagu": &graphql.Field{

				Type: graphql.Boolean,

				Args: graphql.FieldConfigArgument{

					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},

				Resolve: func(p graphql.ResolveParams) (interface{}, error) {

					id := p.Args["id"].(int)

					result, err := db.Exec(
						context.Background(),
						`
						DELETE FROM lagu
						WHERE id = $1
						`,
						id,
					)

					if err != nil {
						return nil, err
					}

					return result.RowsAffected() > 0, nil
				},
			},
		},
	})

	// ========================================================
	// CREATE SCHEMA
	// ========================================================

	return graphql.NewSchema(
		graphql.SchemaConfig{
			Query:    queryType,
			Mutation: mutationType,
		},
	)
}

// ============================================================
// GRAPHQL HANDLER
// ============================================================

func graphqlHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet &&
		r.Method != http.MethodPost {

		errorResponse(
			w,
			http.StatusMethodNotAllowed,
			"Method tidak diperbolehkan",
		)

		return
	}

	schema, err := createGraphQLSchema()

	if err != nil {

		errorResponse(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)

		return
	}

	var request struct {
		Query         string                 `json:"query"`
		Variables     map[string]interface{} `json:"variables"`
		OperationName string                 `json:"operationName"`
	}

	// POST
	if r.Method == http.MethodPost {

		err := json.NewDecoder(r.Body).Decode(&request)

		if err != nil {

			errorResponse(
				w,
				http.StatusBadRequest,
				"Format JSON tidak valid",
			)

			return
		}
	}

	// GET
	if r.Method == http.MethodGet {
		request.Query = r.URL.Query().Get("query")
	}

	// Query kosong
	if request.Query == "" {

		errorResponse(
			w,
			http.StatusBadRequest,
			"Query GraphQL wajib diisi",
		)

		return
	}

	// ========================================================
	// RESET COUNTER
	// ========================================================

	queryPenyanyiCallCount = 0
	resolverCallCount = 0

	// ========================================================
	// EXECUTE GRAPHQL
	// ========================================================

	result := graphql.Do(
		graphql.Params{
			Schema:         schema,
			RequestString:  request.Query,
			VariableValues: request.Variables,
			OperationName:  request.OperationName,
		},
	)

	// ========================================================
	// TAMPILKAN HASIL N+1
	// ========================================================

	if queryPenyanyiCallCount > 0 {

		log.Printf(
			"Query.penyanyi dipanggil: %d kali",
			queryPenyanyiCallCount,
		)

		log.Printf(
			"Resolver Penyanyi.album dipanggil: %d kali",
			resolverCallCount,
		)

		log.Printf(
			"Prediksi N+1: 1 + %d = %d",
			resolverCallCount,
			1+resolverCallCount,
		)
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	_ = json.NewEncoder(w).Encode(result)
}

// ============================================================
// GRAPHQL ROUTE + CORS
// ============================================================

func graphqlRoute() http.Handler {

	return cors.New(
		cors.Options{

			AllowedOrigins: []string{
				"https://studio.apollographql.com",
			},

			AllowedMethods: []string{
				"GET",
				"POST",
				"OPTIONS",
			},

			AllowedHeaders: []string{
				"Content-Type",
				"Accept",
				"Origin",
				"Apollo-Require-Preflight",
				"X-Apollo-Operation-Name",
				"Apollographql-Client-Name",
				"Apollographql-Client-Version",
			},

			AllowCredentials: false,
		},
	).Handler(
		http.HandlerFunc(graphqlHandler),
	)
}