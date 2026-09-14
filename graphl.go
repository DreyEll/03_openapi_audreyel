package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/graphql-go/graphql"
	"github.com/rs/cors"
)

// ============================================================
// CREATE GRAPHQL SCHEMA
// ============================================================

func createGraphQLSchema() (graphql.Schema, error) {

	// ========================================================
	// PENYANYI TYPE
	// ========================================================

	penyanyiType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Penyanyi",

		Fields: graphql.Fields{
			"nama": &graphql.Field{
				Type: graphql.String,
			},
		},
	})

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

			// Relasi:
			// album.penyanyi_id -> penyanyi.id
			"penyanyi": &graphql.Field{
				Type: penyanyiType,

				Resolve: func(p graphql.ResolveParams) (interface{}, error) {

					album, ok := p.Source.(map[string]interface{})
					if !ok {
						return nil, nil
					}

					penyanyiID, ok := album["penyanyi_id"].(int)
					if !ok {
						return nil, nil
					}

					var nama string

					err := db.QueryRow(
						context.Background(),
						`
						SELECT nama
						FROM penyanyi
						WHERE id = $1
						`,
						penyanyiID,
					).Scan(&nama)

					if err != nil {
						return nil, err
					}

					return map[string]interface{}{
						"nama": nama,
					}, nil
				},
			},
		},
	})

	// ========================================================
	// LAGU TYPE
	// ========================================================

	laguType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Lagu",

		Fields: graphql.Fields{

			"judul": &graphql.Field{
				Type: graphql.String,
			},

			// Relasi:
			// lagu.album_id -> album.id
			"album": &graphql.Field{
				Type: albumType,

				Resolve: func(p graphql.ResolveParams) (interface{}, error) {

					lagu, ok := p.Source.(map[string]interface{})
					if !ok {
						return nil, nil
					}

					albumID, ok := lagu["album_id"].(int)
					if !ok {
						return nil, nil
					}

					var namaAlbum string
					var tahun int
					var penyanyiID int

					err := db.QueryRow(
						context.Background(),
						`
						SELECT nama_album, tahun, penyanyi_id
						FROM album
						WHERE id = $1
						`,
						albumID,
					).Scan(
						&namaAlbum,
						&tahun,
						&penyanyiID,
					)

					if err != nil {
						return nil, err
					}

					return map[string]interface{}{
						"nama_album":  namaAlbum,
						"tahun":       tahun,
						"penyanyi_id": penyanyiID,
					}, nil
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
			// LAGU
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
	// CREATE SCHEMA
	// ========================================================

	return graphql.NewSchema(
		graphql.SchemaConfig{
			Query: queryType,
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

	// Execute GraphQL
	result := graphql.Do(
		graphql.Params{
			Schema:         schema,
			RequestString:  request.Query,
			VariableValues: request.Variables,
			OperationName:  request.OperationName,
		},
	)

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