package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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

// Get post comments
func (s *Service) GetPostComments(ctx context.Context, postId int64, last int, before string) ([]Comment, error) {
	var comments []Comment
	uid, auth := ctx.Value(KeyAuthUserID).(int64)

	query, args, err := buildQuery(`SELECT comments.id, comments.user_id, comments.post_id, comments.content, comments.likes_count,comments.created_at
	,users.username As username, users.avatar As avatar_url
	{{if .Auth}}
	,comments.user_id = @uid As mine
	,comment_likes.user_id is not null As liked
	{{end}}
	FROM comments
	Inner join users on users.id = comments.user_id
	{{if .Auth}}
	LEFT JOIN comment_likes ON comment_likes.comment_id = comments.id AND comment_likes.user_id = @uid
	{{end}}
	WHERE comments.post_id = @postId
	{{if .before}}
	AND comments.id < @before
	{{end}}
	order by comments.id desc
	{{if .last}}
	limit @last
	{{end}}	
	`, map[string]interface{}{
		"last":   last,
		"before": before,
		"postId": postId,
		"uid":    uid,
		"Auth":   auth,
	})
	if err != nil {
		return comments, fmt.Errorf("can not build comments query, error: %v", err)
	}
	var u User
	var avatar sql.NullString
	rows, err := s.Db.Query(ctx, query, args...)
	if isforeignKeyViolation(err) {
		return comments, ErrPostNotFound
	}

	defer rows.Close()

	if err != nil {
		return comments, fmt.Errorf("can not get comments, error: %v", err)
	}

	for rows.Next() {
		var comment Comment
		dest := []interface{}{&comment.ID, &comment.UserId, &comment.PostId, &comment.Content, &comment.LikesCount, &comment.CreatedAt, &u.Username, &avatar}
		if auth {
			dest = append(dest, &comment.IsMe, &comment.Liked)
		}
		if err = rows.Scan(dest...); err != nil {
			return comments, fmt.Errorf("can not scan comment, error: %v", err)
		}
		if avatar.Valid {
			url := s.Origin + "/img/avatars" + avatar.String
			u.AvatarUrl = &url
		}
		comment.User = &u
		comments = append(comments, comment)

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return comments, err
}
