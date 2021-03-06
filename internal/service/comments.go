package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

type ToggleCommentLikeOutput struct {
	Liked      bool `json:"liked"`
	LikesCount int  `json:"likes_count"`
}

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

var (
	ErrCommentNotFound = errors.New("comment not found")
)

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

	//subscripe the use to post_subscription
	query = "INSERT INTO post_subscriptions (user_id, post_id) VALUES ($1, $2) ON CONFLICT(user_id,post_id) DO NOTHING"
	if _, err = tx.Exec(ctx, query, uid, postId); err != nil {
		return comment, fmt.Errorf("can not subscripe the user to post_subscription after creating comment, error: %v", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return comment, fmt.Errorf("can not commit the creating comment transcation, error: %v", err)
	}

	go s.CommentCreated(comment)
	return comment, nil
}

func (s *Service) CommentCreated(c Comment) {
	//get user detals
	u, err := s.UserById(context.Background(), c.UserId)
	if err != nil {
		log.Printf("can not get user details, error: %v", err)
		return
	}
	c.User = &u
	c.IsMe = false
	go s.NotifyComment(c)
	go s.NotifyCommentMention(c)

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

//toggle comments like and update comment likes count
func (s *Service) ToggleCommentLike(ctx context.Context, commentId int64) (ToggleCommentLikeOutput, error) {
	var output ToggleCommentLikeOutput
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return output, ErrUnAuthorized
	}
	//Begin transasction
	tx, err := s.Db.Begin(ctx)
	if err != nil {
		return output, fmt.Errorf("can not start the creating comment transcation, error: %v", err)
	}
	defer tx.Rollback(ctx)

	//query to check if the user has liked the comment
	query := "Select Exists (SELECT 1 FROM comment_likes WHERE comment_id = $1 AND user_id = $2)"

	if err := tx.QueryRow(ctx, query, commentId, uid).Scan(&output.Liked); err != nil {
		return output, fmt.Errorf("can not check if the user has liked the comment, error: %v", err)
	}

	if output.Liked {

		//if user already liked the comment delete the like
		query = "DELETE FROM comment_likes WHERE comment_id = $1 AND user_id = $2"
		if _, err = tx.Exec(ctx, query, commentId, uid); err != nil {
			return output, fmt.Errorf("can not delete the like, error: %v", err)
		}
		//update comment likes count
		query = "UPDATE comments SET likes_count = likes_count - 1 WHERE id = $1 Returning likes_count"
		if err = tx.QueryRow(ctx, query, commentId).Scan(&output.LikesCount); err != nil {
			return output, fmt.Errorf("can not update comment likes count in negative, error: %v", err)
		}

	} else {
		//if user didnt liked the comment insert the like
		query = "INSERT INTO comment_likes (comment_id, user_id) VALUES ($1, $2)"
		_, err = tx.Exec(ctx, query, commentId, uid)

		if isforeignKeyViolation(err) {
			return output, ErrCommentNotFound
		}

		if err != nil {
			return output, fmt.Errorf("can not insert the like, error: %v", err)
		}
		//update comment likes count
		query = "UPDATE comments SET likes_count = likes_count + 1 WHERE id = $1 Returning likes_count"
		if err = tx.QueryRow(ctx, query, commentId).Scan(&output.LikesCount); err != nil {
			return output, fmt.Errorf("can not update comment likes count in postive, error: %v", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return output, fmt.Errorf("can not commit the creating comment transcation, error: %v", err)
	}
	output.Liked = !output.Liked

	return output, nil
}
