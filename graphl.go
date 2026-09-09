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
	// ALBUM TYPE
	// ========================================================

	albumType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Album",

		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type: graphql.String,
			},

			"year": &graphql.Field{
				Type: graphql.Int,
			},
		},
	})

	// ========================================================
	// PRODUCT TYPE
	// products = data dari tabel lagu
	// name = judul lagu
	// album = relasi ke tabel album
	// ========================================================

	productType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Product",

		Fields: graphql.Fields{

			// ------------------------------------------------
			// name
			// GraphQL: name
			// Database: lagu.judul
			// ------------------------------------------------

			"name": &graphql.Field{
				Type: graphql.String,

				Resolve: func(p graphql.ResolveParams) (interface{}, error) {

					product, ok := p.Source.(map[string]interface{})

					if !ok {
						return nil, nil
					}

					return product["name"], nil
				},
			},

			// ------------------------------------------------
			// album
			// Relasi:
			// lagu.album_id -> album.id
			// ------------------------------------------------

			"album": &graphql.Field{
				Type: albumType,

				Resolve: func(p graphql.ResolveParams) (interface{}, error) {

					product, ok := p.Source.(map[string]interface{})

					if !ok {
						return nil, nil
					}

					albumID, ok := product["album_id"].(int)

					if !ok {
						return nil, nil
					}

					var namaAlbum string
					var tahun int

					err := db.QueryRow(
						context.Background(),
						`
						SELECT nama_album, tahun
						FROM album
						WHERE id = $1
						`,
						albumID,
					).Scan(
						&namaAlbum,
						&tahun,
					)

					if err != nil {
						return nil, err
					}

					return map[string]interface{}{
						"name": namaAlbum,
						"year": tahun,
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

			"products": &graphql.Field{
				Type: graphql.NewList(productType),

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

					products := []interface{}{}

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

						products = append(
							products,
							map[string]interface{}{
								"id":       id,
								"name":     judul,
								"album_id": albumID,
							},
						)
					}

					if err := rows.Err(); err != nil {
						return nil, err
					}

					return products, nil
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

	// Hanya menerima GET dan POST.
	if r.Method != http.MethodGet &&
		r.Method != http.MethodPost {

		errorResponse(
			w,
			http.StatusMethodNotAllowed,
			"Method tidak diperbolehkan",
		)

		return
	}

	// --------------------------------------------------------
	// Buat GraphQL schema
	// --------------------------------------------------------

	schema, err := createGraphQLSchema()

	if err != nil {

		errorResponse(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)

		return
	}

	// --------------------------------------------------------
	// Request GraphQL
	// --------------------------------------------------------

	var request struct {
		Query         string                 `json:"query"`
		Variables     map[string]interface{} `json:"variables"`
		OperationName string                 `json:"operationName"`
	}

	// --------------------------------------------------------
	// POST
	// --------------------------------------------------------

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

	// --------------------------------------------------------
	// GET
	// --------------------------------------------------------

	if r.Method == http.MethodGet {
		request.Query = r.URL.Query().Get("query")
	}

	// --------------------------------------------------------
	// Query kosong
	// --------------------------------------------------------

	if request.Query == "" {

		errorResponse(
			w,
			http.StatusBadRequest,
			"Query GraphQL wajib diisi",
		)

		return
	}

	// --------------------------------------------------------
	// Execute GraphQL
	// --------------------------------------------------------

	result := graphql.Do(
		graphql.Params{
			Schema:         schema,
			RequestString:  request.Query,
			VariableValues: request.Variables,
			OperationName:  request.OperationName,
		},
	)

	// --------------------------------------------------------
	// Response
	// --------------------------------------------------------

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

			// Apollo Sandbox
			AllowedOrigins: []string{
				"https://studio.apollographql.com",
			},

			// GraphQL membutuhkan GET dan POST.
			// OPTIONS digunakan untuk CORS preflight.
			AllowedMethods: []string{
				"GET",
				"POST",
				"OPTIONS",
			},

			// Header yang dapat dikirim Apollo Sandbox.
			AllowedHeaders: []string{
				"Content-Type",
				"Accept",
				"Origin",
				"Apollo-Require-Preflight",
				"X-Apollo-Operation-Name",
				"Apollographql-Client-Name",
				"Apollographql-Client-Version",
			},

			// Mengizinkan browser menerima response.
			AllowCredentials: false,
		},
	).Handler(
		http.HandlerFunc(graphqlHandler),
	)
}
