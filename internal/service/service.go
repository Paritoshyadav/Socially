package service

import (
	"github.com/jackc/pgx/v4"
	"github.com/paritoshyadav/socialnetwork/internal/service/codec"
)

//logics
type Service struct {
	Db     *pgx.Conn
	Codec  codec.CodecLayer
	Origin string
}

func New(db *pgx.Conn, codec codec.CodecLayer, origin string) *Service {
	return &Service{
		Db:     db,
		Codec:  codec,
		Origin: origin,
	}
}
