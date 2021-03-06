package service

import (
	"sync"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/paritoshyadav/socialnetwork/internal/service/codec"
)

//logics
type Service struct {
	Db                  *pgxpool.Pool
	Codec               codec.CodecLayer
	Origin              string
	timelineITemClients sync.Map
}

func New(db *pgxpool.Pool, codec codec.CodecLayer, origin string) *Service {
	return &Service{
		Db:     db,
		Codec:  codec,
		Origin: origin,
	}
}
