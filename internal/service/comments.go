package service

import (
	"context"
	"fmt"
	"time"
)

//comment struct
type Comment struct {
	ID         int64     `json:"id"`
	UserId     int64     `json:"-"`
	PostId     int64     `json:"-"`
	Content    string    `json:"content"`
	LikesCount int       `json:"likes_count"`
	Liked      bool      `json:"liked"`
	User       *User     `json:"user ,omitempty"`
	IsMe       bool      `json:"is_me"`
	CreatedAt  time.Time `json:"created_at"`
}

//CreateComment and update post comment Count
func (s *Service) CreateComment(ctx context.Context, content string, postId int64) (Comment, error) {
	var comment Comment
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return comment, ErrUnAuthorized
	}
	//Begin transasction
	tx, err := s.Db.Begin(ctx)
	if err != nil {
		return comment, fmt.Errorf("can not start the creating comment transcation, error: %v", err)
	}
	defer tx.Rollback(ctx)
	//query to create comment and get the comment id,created_at,updated_at
	query := "INSERT INTO comments (user_id, post_id, content) VALUES ($1, $2, $3) RETURNING id, created_at"

	err = tx.QueryRow(ctx, query, uid, postId, content).Scan(&comment.ID, &comment.CreatedAt)
	if isforeignKeyViolation(err) {
		return comment, ErrPostNotFound
	}

	if err != nil {
		return comment, fmt.Errorf("can not insert comment, error: %v", err)
	}

	comment.UserId = uid
	comment.PostId = postId
	comment.Content = content
	comment.IsMe = true

	//update post comment count
	query = "UPDATE posts SET comments_count = comments_count + 1 WHERE id = $1"
	if _, err = tx.Exec(ctx, query, postId); err != nil {
		return comment, fmt.Errorf("can not update post comment count, error: %v", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return comment, fmt.Errorf("can not commit the creating comment transcation, error: %v", err)
	}

	return comment, nil
}
