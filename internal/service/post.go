package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sanity-io/litter"
)

//Post model
type Post struct {
	ID            int64     `json:"id"`
	UserId        int64     `json:"-"`
	Content       string    `json:"content"`
	LikesCount    int       `json:"likes_count"`
	CommentsCount int       `json:"comments_count"`
	Liked         bool      `json:"liked"`
	SpoilerOf     *string   `json:"spoiler_of"`
	NSFW          bool      `json:"nsfw"`
	User          *User     `json:"user"`
	IsMe          bool      `json:"is_me"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Subscribed    bool      `json:"subscribed"`
}

var (
	ErrPostNotFound = errors.New("post not found")
)

type TogglePostLikeOutput struct {
	Liked      bool `json:"liked"`
	LikedCount int  `json:"liked_count"`
}

// CreatePost creates a new post and add to timeline.

func (s *Service) CreatePost(ctx context.Context, content string, spoilerOf *string, nsfw bool) (TimelineItem, error) {
	var ti TimelineItem
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return ti, ErrUnAuthorized
	}
	//Begin transasction
	tx, err := s.Db.Begin(ctx)
	if err != nil {
		return ti, fmt.Errorf("can not start the creating post transcation, error: %v", err)
	}
	defer tx.Rollback(ctx)
	//query to create post and get the post id,created_at,updated_at
	query := "INSERT INTO posts (user_id, content, spoiler, nsfw) VALUES ($1, $2, $3, $4) RETURNING id, created_at ,updated_at"

	if err = tx.QueryRow(ctx, query, uid, content, spoilerOf, nsfw).Scan(&ti.Post.ID, &ti.Post.CreatedAt, &ti.Post.UpdatedAt); err != nil {
		return ti, fmt.Errorf("can not insert post, error: %v", err)
	}

	ti.Post.UserId = uid
	ti.Post.Content = content
	ti.Post.SpoilerOf = spoilerOf
	ti.Post.NSFW = nsfw
	ti.Post.IsMe = true

	//query to subscribe user
	query = "INSERT INTO post_subscriptions (user_id, post_id) VALUES ($1, $2)"
	if _, err = tx.Exec(ctx, query, uid, ti.Post.ID); err != nil {
		return ti, fmt.Errorf("can not subscribe user, error: %v", err)
	}

	ti.Post.Subscribed = true

	//query to add post to timeline returning id

	query = "INSERT INTO timelines (user_id, post_id) VALUES ($1, $2) RETURNING id"

	if err = tx.QueryRow(ctx, query, uid, ti.Post.ID).Scan(&ti.ID); err != nil {
		return ti, fmt.Errorf("can not insert post to timeline, error: %v", err)
	}

	ti.UserId = uid
	ti.PostId = ti.Post.ID

	//commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return ti, fmt.Errorf("can not commit the creating post transcation, error: %v", err)
	}

	go func(p Post) {

		u, err := s.UserById(context.Background(), p.UserId)
		if err != nil {
			log.Printf("can not get post user by id: %v", err)
			return
		}

		p.User = &u
		p.IsMe = false
		p.Subscribed = false

		ti, err := s.fanoutPost(p)
		if err != nil {
			log.Printf("can not fanout post: %v", err)
			return
		}

		for _, t := range ti {
			log.Println(litter.Sdump(t))
			//TODO:s.PublishPost(t)
		}

	}(ti.Post)
	log.Println(litter.Sdump(ti))
	return ti, nil
}

func (s *Service) fanoutPost(p Post) ([]TimelineItem, error) {
	query := "Insert into timelines (user_id, post_id) select follower_id, $1 from follows where following_id = $2 RETURNING id, user_id"
	rows, err := s.Db.Query(context.Background(), query, p.ID, p.UserId)
	if err != nil {
		return nil, fmt.Errorf("can not fanout post, error: %v", err)
	}
	defer rows.Close()
	var ti []TimelineItem
	for rows.Next() {
		var t TimelineItem
		if err = rows.Scan(&t.ID, &t.UserId); err != nil {
			return nil, fmt.Errorf("can not scan fanout post, error: %v", err)
		}
		t.PostId = p.ID
		t.Post = p
		ti = append(ti, t)
	}
	//rows error
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("can not iterate timeline post, error: %v", err)
	}

	return ti, nil
}

//toggle posts likes

func (s *Service) TogglePostLike(ctx context.Context, postId int64) (TogglePostLikeOutput, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	var tpl TogglePostLikeOutput
	if !ok {
		return tpl, ErrUnAuthorized
	}
	//Begin transasction
	tx, err := s.Db.Begin(ctx)
	if err != nil {
		return tpl, fmt.Errorf("can not start the post likke transcation, error: %v", err)
	}
	defer tx.Rollback(ctx)
	//query to check if user liked the post
	query := "SELECT EXISTS (SELECT 1 FROM likes WHERE user_id = $1 AND post_id = $2)"
	err = tx.QueryRow(ctx, query, uid, postId).Scan(&tpl.Liked)
	if err != nil {
		return tpl, fmt.Errorf("can not check if user liked the post, error: %v", err)
	}

	//query to toggle likes
	if tpl.Liked {
		//User already liked the post, so unlike it
		query = "DELETE FROM likes WHERE user_id = $1 AND post_id = $2"
		_, err = tx.Exec(ctx, query, uid, postId)
		if err != nil {
			return tpl, fmt.Errorf("can not unlike the post, error: %v", err)
		}
		//update post likes count eurning likes count
		query = "UPDATE posts SET likes_count = likes_count - 1 WHERE id = $1 RETURNING likes_count"
		err = tx.QueryRow(ctx, query, postId).Scan(&tpl.LikedCount)
		if err != nil {
			return tpl, fmt.Errorf("can not update post likes count negative, error: %v", err)
		}

	} else {
		//User not liked the post, so like it
		query = "INSERT INTO likes (user_id, post_id) VALUES ($1, $2)"
		_, err := tx.Exec(ctx, query, uid, postId)
		// Forirgn key constraint violation
		if isforeignKeyViolation(err) {
			return tpl, ErrPostNotFound
		}
		if err != nil {
			return tpl, fmt.Errorf("can not like the post, error: %v", err)
		}

		//update post likes count reurning likes count
		query = "UPDATE posts SET likes_count = likes_count + 1 WHERE id = $1 RETURNING likes_count"
		err = tx.QueryRow(ctx, query, postId).Scan(&tpl.LikedCount)
		if err != nil {
			return tpl, fmt.Errorf("can not update post likes count postive, error: %v", err)
		}

	}
	//commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return tpl, fmt.Errorf("can not commit the post likke transcation, error: %v", err)
	}

	tpl.Liked = !tpl.Liked
	return tpl, nil
}

//get posts by user username
func (s *Service) PostsByUser(ctx context.Context, username string, last int, before string) ([]Post, error) {
	uid, auth := ctx.Value(KeyAuthUserID).(int64)

	var posts []Post
	query, args, err := buildQuery(`SELECT id,content, created_at,likes_count,spoiler,nsfw,comments_count
	{{if .auth}}
	,posts.user_id = @uid As mine
	,likes.user_id is not null As liked
	,post_subscriptions.user_id is not null As subscribed
	{{end}}
	FROM posts 	
	{{if .auth}}
	LEFT JOIN likes ON likes.post_id = posts.id AND likes.user_id = @uid
	LEFT JOIN post_subscriptions ON post_subscriptions.post_id = posts.id AND post_subscriptions.user_id = @uid	
	{{end}}
	WHERE posts.user_id = (SELECT id FROM users WHERE username = @username)
	{{if .before}}
	AND posts.id < @before
	{{end}}
	order by posts.id desc
	{{if .last}}
	limit @last	
	{{end}}
	`, map[string]interface{}{
		"auth":     auth,
		"username": username,
		"uid":      uid,
		"last":     last,
		"before":   before,
	})
	if err != nil {
		return posts, fmt.Errorf("can not build posts query, error: %v", err)
	}
	rows, err := s.Db.Query(ctx, query, args...)
	if err != nil {
		return posts, fmt.Errorf("can not get posts, error: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p Post
		dest := []interface{}{&p.ID, &p.Content, &p.CreatedAt, &p.LikesCount, &p.SpoilerOf, &p.NSFW, &p.CommentsCount}
		if auth {
			dest = append(dest, &p.IsMe, &p.Liked, &p.Subscribed)
		}

		if err = rows.Scan(dest...); err != nil {
			return posts, fmt.Errorf("can not scan post, error: %v", err)
		}
		posts = append(posts, p)
	}
	return posts, nil
}

// get post by id
func (s *Service) Post(ctx context.Context, id int64) (Post, error) {
	uid, auth := ctx.Value(KeyAuthUserID).(int64)

	var p Post
	query, args, err := buildQuery(`SELECT posts.id,content, created_at,likes_count,spoiler,nsfw,comments_count 
	,users.username As username
	,users.avatar As avatar_url
	{{if .auth}}
	,posts.user_id = @uid As mine
	,likes.user_id is not null As liked
	,post_subscriptions.user_id is not null As subscribed
	
	{{end}}
	FROM posts 
	Inner join users on users.id = posts.user_id	
	{{if .auth}}
	
	LEFT JOIN likes ON likes.post_id = posts.id AND likes.user_id = @uid
	LEFT JOIN post_subscriptions ON post_subscriptions.post_id = posts.id AND post_subscriptions.user_id = @uid	
	{{end}}
	WHERE posts.id = @id
	order by posts.id desc	
	`, map[string]interface{}{
		"auth": auth,
		"id":   id,
		"uid":  uid,
	})
	if err != nil {
		return p, fmt.Errorf("can not build posts query, error: %v", err)
	}
	var u User
	var avatar sql.NullString
	dest := []interface{}{&p.ID, &p.Content, &p.CreatedAt, &p.LikesCount, &p.SpoilerOf, &p.NSFW, &p.CommentsCount, &u.Username, &avatar}
	if auth {

		dest = append(dest, &p.IsMe, &p.Liked, &p.Subscribed)
	}
	err = s.Db.QueryRow(ctx, query, args...).Scan(dest...)
	if err != nil {
		return p, fmt.Errorf("can not get posts, error: %v", err)
	}
	if avatar.Valid {
		url := s.Origin + "/img/avatars" + avatar.String
		u.AvatarUrl = &url
	}
	p.User = &u

	return p, nil
}
