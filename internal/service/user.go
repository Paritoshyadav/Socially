package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v4"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

var (
	ErrUserNotFound           = errors.New("User Not Found")
	ErrNoRowAffected          = errors.New("no row afftected")
	ErrEmailTaken             = errors.New("email already taken")
	ErrUsernameTaken          = errors.New("username already taken")
	ErrInvalidFollow          = errors.New("can not follow yourself")
	ErrInvalidUsername        = errors.New("invalid email address")
	ErrValidations            = errors.New("validations Fail")
	ErrUnSpportedAvatarFormat = errors.New("wrong Image format")
)

var (
	avatarsDir = path.Join("web", "static", "img", "avatars")
)

const MaxAvatarBytes = 5 << 20 //5MB

//User model
type User struct {
	ID        int64   `json:"id,omitempty" validate:"required"`
	Username  string  `json:"username,omitempty" validate:"required,email"`
	AvatarUrl *string `json:"avatarUrl"` //using pointer as it can be null
}

type ToggleFollowOutput struct {
	Following      bool `json:"following"`
	FollowersCount int  `json:"followers_count"`
}

type UserProfile struct {
	User
	Email           string `json:"email,omitempty"`
	FollowersCount  int    `json:"followers_count"`
	FollowingsCount int    `json:"following_count"`
	Me              bool   `json:"me,omitempty"`
	Following       bool   `json:"following"` //TODO: instead of bool get list nfollowing user and follower user and check if user id contain in the list
	FollowingBack   bool   `json:"following_back"`
}

func (u User) Validate() error {
	validate := validator.New()
	return validate.Struct(u)

}

func (s *Service) CreateUser(ctx context.Context, email string, username string) error {
	query := "INSERT INTO users (email, username) VALUES ($1,$2)"
	commandTag, err := s.Db.Exec(ctx, query, email, username)

	fmt.Printf("users")
	ok := isUnquieViolation(err)
	fmt.Printf("is unique %v %v", err, ok)
	if ok && strings.Contains(err.Error(), "email") {
		return ErrEmailTaken
	}
	if ok && strings.Contains(err.Error(), "username") {
		return ErrUsernameTaken
	}
	if commandTag.RowsAffected() != 1 {
		return ErrNoRowAffected
	}
	if err != nil {
		return err
	}
	return nil

}

//func to get user by id

func (s *Service) UserById(ctx context.Context, uid int64) (User, error) {
	var u User
	var avatar sql.NullString

	query := "select username,avatar from users where id = $1"
	err := s.Db.QueryRow(ctx, query, uid).Scan(&u.Username, &avatar)
	if err == pgx.ErrNoRows {
		return u, ErrUserNotFound
	}
	if err != nil {
		return u, fmt.Errorf("unable to query: %v", err)
	}
	u.ID = uid
	if avatar.Valid {
		url := s.Origin + "img/avatars" + avatar.String
		u.AvatarUrl = &url
	}
	return u, nil

}

func (s *Service) ToggleFollow(ctx context.Context, username string) (ToggleFollowOutput, error) {
	var out ToggleFollowOutput
	userId, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return out, ErrUnAuthorized
	}
	tx, err := s.Db.Begin(ctx)
	defer tx.Rollback(ctx)
	if err != nil {
		return out, fmt.Errorf("can not start the transcation, error: %v", err)
	}
	query := "select id from users where username = $1"
	var followingID int64
	err = tx.QueryRow(ctx, query, username).Scan(&followingID)
	if err == pgx.ErrNoRows {
		return out, ErrUserNotFound
	}
	if err != nil {
		return out, fmt.Errorf("could not query id from given username, error %v", err)
	}

	if userId == followingID {
		return out, ErrInvalidFollow
	}
	query = "select exists (select 1 from follows where follower_id = $1 and following_id = $2)"
	if err = tx.QueryRow(ctx, query, userId, followingID).Scan(&out.Following); err != nil {
		return out, fmt.Errorf("could not query select existance of following user: %v", err)
	}
	fmt.Println(out)
	if out.Following {
		query = "DELETE FROM follows where follower_id = $1 and following_id = $2"
		if _, err = tx.Exec(ctx, query, userId, followingID); err != nil {
			return out, fmt.Errorf("could not delete follow ,%v", err)
		}
		query = "UPDATE users SET followings_count = followings_count -1 WHERE id = $1"
		if _, err = tx.Exec(ctx, query, userId); err != nil {
			return out, fmt.Errorf("could to update user following count,%v", err)
		}
		query = "UPDATE users SET followers_count = followers_count -1 WHERE id = $1 RETURNING followers_count"
		if err = tx.QueryRow(ctx, query, followingID).Scan(&out.FollowersCount); err != nil {
			return out, fmt.Errorf("could to update following user followers count,%v", err)
		}
	} else {
		query = "INSERT INTO follows(follower_id,following_id) VALUES($1,$2)"
		if _, err = tx.Exec(ctx, query, userId, followingID); err != nil {
			return out, fmt.Errorf("could not delete follow ,%v", err)
		}
		query = "UPDATE users SET followings_count = followings_count + 1 WHERE id = $1"
		if _, err = tx.Exec(ctx, query, userId); err != nil {
			return out, fmt.Errorf("could to update user following count,%v", err)
		}
		query = "UPDATE users SET followers_count = followers_count + 1 WHERE id = $1 RETURNING followers_count"
		if err = tx.QueryRow(ctx, query, followingID).Scan(&out.FollowersCount); err != nil {
			return out, fmt.Errorf("could to update following user followers count,%v", err)
		}

	}
	err = tx.Commit(context.Background())
	if err != nil {
		return out, fmt.Errorf("could not commit follow toggle transcation, %v", err)
	}
	out.Following = !out.Following
	fmt.Println(out)
	//TODO: notify the user about following
	// if out.Following {

	// }
	return out, nil

}

func (s *Service) User(ctx context.Context, username string) (UserProfile, error) {

	var profile UserProfile
	userID, auth := ctx.Value(KeyAuthUserID).(int64)
	query := "SELECT id,email,followers_count,followings_count "
	args := []interface{}{username}
	dest := []interface{}{&profile.ID, &profile.Email, &profile.FollowersCount, &profile.FollowingsCount}
	if auth {
		query += ","
		query += "following.following_id IS NOT NULL AS following,"
		query += "followingback.follower_id IS NOT NULL AS followingback "

		dest = append(dest, &profile.Following, &profile.FollowingBack)
	}
	query += "FROM users "
	if auth {
		query += "LEFT JOIN follows AS following ON following.following_id = users.id and following.follower_id = $2 "
		query += "LEFT JOIN follows AS followingback ON followingback.following_id = $2 and followingback.follower_id = users.id "
		args = append(args, userID)
	}

	query += "where username = $1"
	err := s.Db.QueryRow(ctx, query, args...).Scan(dest...)
	if err == pgx.ErrNoRows {
		return profile, ErrUserNotFound
	}
	if err != nil {
		return UserProfile{}, fmt.Errorf("not able to query selected user, %v", err)
	}

	profile.Me = auth && userID == profile.ID
	if !profile.Me {
		profile.ID = 0
		profile.Email = ""
		profile.Me = false

	}
	profile.Username = username

	return profile, nil
}

func (s *Service) Users(ctx context.Context, search string, first int, after string) ([]UserProfile, error) {
	uid, auth := ctx.Value(KeyAuthUserID).(int64)
	first = normalizePageSize(first)
	search = strings.TrimSpace(search)
	after = strings.TrimSpace(after)
	query, args, err := buildQuery(`SELECT id,email,avatar,username,followers_count,followings_count
	{{if .auth}}
	,following.following_id IS NOT NULL AS following
	,followingback.follower_id IS NOT NULL AS followingback
	{{end}} 
	FROM users 
	{{if .auth}}
	LEFT JOIN follows AS following ON following.follower_id = @uid AND following.following_id =users.id
	LEFT JOIN follows AS followingback ON followingback.following_id =users.id AND followingback.follower_id = @uid
	{{end}}	
	{{if or .search .after}}WHERE{{end}}
	{{if .search}}
	username ILIKE '%' || @search || '%'
	{{end}}
	{{if and .search .after}} AND {{end}}
	{{if .after}} username > @after {{end}}	
	ORDER BY username ASC
	LIMIT @first
	`, map[string]interface{}{
		"auth":   auth,
		"search": search,
		"uid":    uid,
		"after":  after,
		"first":  first,
	})
	if err != nil {
		return nil, err
	}
	fmt.Println(query)
	rows, err := s.Db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var avatar sql.NullString
	uu := make([]UserProfile, 0, first)

	for rows.Next() {
		var profile UserProfile
		dest := []interface{}{&profile.ID, &profile.Email, &avatar, &profile.Username, &profile.FollowersCount, &profile.FollowingsCount}

		if auth {
			dest = append(dest, &profile.Following, &profile.FollowingBack)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan user profile: %v", err)
		}

		profile.Me = auth && uid == profile.ID
		if !profile.Me {
			profile.ID = 0
			profile.Email = ""
			profile.Me = false

		}

		if avatar.Valid {
			url := s.Origin + "/img/avatars" + avatar.String
			profile.AvatarUrl = &url
		}

		uu = append(uu, profile)

	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate user rows: %w", err)
	}

	return uu, nil
}

func (s *Service) UserFollowers(ctx context.Context, username string, first int, after string) ([]UserProfile, error) {
	uid, auth := ctx.Value(KeyAuthUserID).(int64)
	first = normalizePageSize(first)
	username = strings.TrimSpace(username)
	after = strings.TrimSpace(after)
	query, args, err := buildQuery(`SELECT id,email,avatar,username,followers_count,followings_count
	{{if .auth}}
	,following.following_id IS NOT NULL AS following
	,followingback.follower_id IS NOT NULL AS followingback
	{{end}} 
	FROM follows
	INNER JOIN users ON users.id = follows.follower_id
	{{if .auth}}
	LEFT JOIN follows AS following ON following.follower_id = @uid AND following.following_id =users.id
	LEFT JOIN follows AS followingback ON followingback.following_id =users.id AND followingback.follower_id = @uid
	{{end}}	
	WHERE follows.following_id = (SELECT id From users where username = @username)
	{{if .after}} AND username > @after {{end}}	
	ORDER BY username ASC
	LIMIT @first
	`, map[string]interface{}{
		"auth":     auth,
		"username": username,
		"uid":      uid,
		"after":    after,
		"first":    first,
	})
	if err != nil {
		return nil, err
	}
	fmt.Println(query)
	rows, err := s.Db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var avatar sql.NullString
	uu := make([]UserProfile, 0, first)

	for rows.Next() {
		var profile UserProfile
		dest := []interface{}{&profile.ID, &profile.Email, &avatar, &profile.Username, &profile.FollowersCount, &profile.FollowingsCount}

		if auth {
			dest = append(dest, &profile.Following, &profile.FollowingBack)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan follower profile: %v", err)
		}

		profile.Me = auth && uid == profile.ID
		if !profile.Me {
			profile.ID = 0
			profile.Email = ""
			profile.Me = false

		}
		if avatar.Valid {
			url := s.Origin + "img/avatars" + avatar.String
			profile.AvatarUrl = &url
		}

		uu = append(uu, profile)

	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate user follower rows: %w", err)
	}

	return uu, nil
}

func (s *Service) UserFollowings(ctx context.Context, username string, first int, after string) ([]UserProfile, error) {
	uid, auth := ctx.Value(KeyAuthUserID).(int64)
	first = normalizePageSize(first)
	username = strings.TrimSpace(username)
	after = strings.TrimSpace(after)
	query, args, err := buildQuery(`SELECT id,email,username,followers_count,followings_count
	{{if .auth}}
	,following.following_id IS NOT NULL AS following
	,followingback.follower_id IS NOT NULL AS followingback
	{{end}} 
	FROM follows
	INNER JOIN users ON users.id = follows.following_id
	{{if .auth}}
	LEFT JOIN follows AS following ON following.follower_id = @uid AND following.following_id =users.id
	LEFT JOIN follows AS followingback ON followingback.following_id =users.id AND followingback.follower_id = @uid
	{{end}}	
	WHERE follows.follower_id = (SELECT id From users where username = @username)
	{{if .after}} AND username > @after {{end}}	
	ORDER BY username ASC
	LIMIT @first
	`, map[string]interface{}{
		"auth":     auth,
		"username": username,
		"uid":      uid,
		"after":    after,
		"first":    first,
	})
	if err != nil {
		return nil, err
	}
	fmt.Println(query)
	rows, err := s.Db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	uu := make([]UserProfile, 0, first)

	for rows.Next() {
		var profile UserProfile
		dest := []interface{}{&profile.ID, &profile.Email, &profile.Username, &profile.FollowersCount, &profile.FollowingsCount}

		if auth {
			dest = append(dest, &profile.Following, &profile.FollowingBack)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan following profile: %v", err)
		}

		profile.Me = auth && uid == profile.ID
		if !profile.Me {
			profile.ID = 0
			profile.Email = ""
			profile.Me = false

		}

		uu = append(uu, profile)

	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate user following rows: %w", err)
	}

	return uu, nil
}

func (s *Service) UpdateAvatar(ctx context.Context, r io.Reader) (string, error) {
	userId, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return "", ErrUnAuthorized
	}
	r = io.LimitReader(r, MaxAvatarBytes)
	img, format, err := image.Decode(r)
	if err != nil {
		return "", fmt.Errorf("could not read the image: %v", err)
	}
	if format != "png" && format != "jpeg" {
		return "", ErrUnSpportedAvatarFormat
	}
	avatar, err := gonanoid.New()
	if err != nil {
		return "", fmt.Errorf("could not generate avatar filename: %v", err)
	}
	if format == "png" {
		avatar += ".png"
	}
	if format == "jpeg" {
		avatar += ".jpeg"

	}
	avatarPath := path.Join(avatarsDir, avatar)
	f, err := os.Create(avatarPath)
	defer func() {
		err = f.Close() //TODO: handling this error better later
	}()
	img = imaging.Fill(img, 400, 400, imaging.Center, imaging.CatmullRom)

	if format == "png" {
		err = png.Encode(f, img)
	}
	if format == "jpeg" {
		err = jpeg.Encode(f, img, nil)

	}
	if err != nil {
		return "", fmt.Errorf("could not encode avatar: %v", err)
	}
	var oldAvatar sql.NullString
	query := `UPDATE users SET avatar = $1 WHERE id = $2 
	RETURNING (SELECT avatar FROM users WHERE id = $2) AS old_avatar`
	if err = s.Db.QueryRow(ctx, query, avatar, userId).Scan(&oldAvatar); err != nil {
		defer os.Remove(avatarPath)
		return "", fmt.Errorf("could not Update avatar: %v", err)
	}

	if oldAvatar.Valid {
		defer os.Remove(path.Join(avatarsDir, oldAvatar.String))
	}

	return s.Origin + "/img/avatars" + avatar, nil
}
