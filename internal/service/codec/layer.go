package codec

import "github.com/hako/branca"

type CodecLayer interface {
	GetToken() *branca.Branca
	EncodeAuthID(id int64) (string, error)
	DecodeAuthID(token string) (int64, error)
}
