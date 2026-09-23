package main

import (
	"context"

	music "music-api/music"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LaguGRPCServer struct {
	music.UnimplementedLaguServiceServer
}

func (s *LaguGRPCServer) GetLagu(
	ctx context.Context,
	req *music.GetLaguRequest,
) (*music.Lagu, error) {

	var lagu music.Lagu

	err := db.QueryRow(
		ctx,
		`SELECT id, judul, album_id FROM lagu WHERE id = $1`,
		req.GetId(),
	).Scan(
		&lagu.Id,
		&lagu.Judul,
		&lagu.AlbumId,
	)

	if err != nil {
		return nil, status.Error(
			codes.NotFound,
			"Lagu tidak ditemukan",
		)
	}

	return &lagu, nil
}