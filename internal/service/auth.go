package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v4"
)

type LoginOutput struct {
	Token      string
	Expiration time.Time
	AuthUser   User
}

const (
	TokenLifetime     = time.Hour * 24 * 14
	KeyAuthUserID key = "auth_user_id"
)

var (
	ErrUnAuthorized = errors.New("UnAuthorized User")
)

type key string

// func (s *Service) AuthUserID(token string) (int64, error) {
// 	fmt.Println("token is  :", token)
// 	decodedToken, err := s.Codec.DecodeToString(token)
// 	if err != nil {
// 		return 0, fmt.Errorf("unable to decode the token, %v", err)
// 	}
// 	convToken, err := strconv.ParseInt(decodedToken, 10, 64)
// 	if err != nil {
// 		return 0, fmt.Errorf("unable to Convert the token, %v", err)
// 	}

// 	return convToken, nil

// }

func (s *Service) Login(ctx context.Context, email string) (LoginOutput, error) {
	var lo LoginOutput
	query := "SELECT ID, email FROM users WHERE email = $1"
	err := s.Db.QueryRow(ctx, query, email).Scan(&lo.AuthUser.ID, &lo.AuthUser.Username)
	if err == pgx.ErrNoRows {
		return lo, ErrUserNotFound
	}
	if err != nil {
		return lo, fmt.Errorf("could not query the user: %v", err)
	}
	fmt.Println(len(strconv.FormatInt(lo.AuthUser.ID, 10)))
	lo.Token, err = s.Codec.EncodeAuthID(lo.AuthUser.ID)
	if err != nil {
		return lo, err
	}

	lo.Expiration = time.Now().Add(TokenLifetime)

	return lo, nil

}

func (s *Service) AuthUser(ctx context.Context) (User, error) {
	var u User
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {

		return u, ErrUnAuthorized

	}

	return s.UserById(ctx, uid)

}
