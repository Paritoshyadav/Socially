package codec

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hako/branca"
)

type Codec struct {
	token         string
	tokenlifetime time.Duration
}

func New(token string, tokenlifetime time.Duration) *Codec {

	return &Codec{token: token, tokenlifetime: tokenlifetime}

}

func (c *Codec) GetToken() *branca.Branca {
	codec := branca.NewBranca(c.token)
	codec.SetTTL(uint32(c.tokenlifetime.Seconds()))
	return codec

}

func (c *Codec) EncodeAuthID(id int64) (string, error) {
	encodeToken, err := c.GetToken().EncodeToString(strconv.FormatInt(id, 10))
	if err != nil {
		return "", fmt.Errorf("failed to encode id to token: %v", err)
	}
	return encodeToken, err

}

func (c *Codec) DecodeAuthID(token string) (int64, error) {
	decodedToken, err := c.GetToken().DecodeToString(token)
	if err != nil {
		return 0, fmt.Errorf("unable to decode the token, %v", err)
	}
	convToken, err := strconv.ParseInt(decodedToken, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to Convert the token, %v", err)
	}

	return convToken, nil

}
