package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/sanity-io/litter"
)

type Notification struct {
	ID        int64     `json:"id"`
	UserId    int64     `json:"-"`
	Type      string    `json:"type"`
	Actors    []string  `json:"actor"`
	Issued_at time.Time `json:"issued_at"`
	Read      bool      `json:"read"`
	PostId    *int64    `json:"post_id,omitempty"`
}

type TogglePostSubscriptionOutput struct {
	Subscribed bool `json:"subscribed"`
}

//Mark all notification read
func (s *Service) MarkNotificationsRead(ctx context.Context) error {
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return ErrUnAuthorized
	}
	query := "UPDATE notifications SET read = true WHERE user_id = $1"
	_, err := s.Db.Exec(ctx, query, uid)
	if err != nil {
		log.Printf("can not mark all notifications read: %v", err)
		return err
	}

	return nil
}

//mark notification read by id
func (s *Service) MarkNotificationRead(ctx context.Context, notificationId int64) error {
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return ErrUnAuthorized
	}
	query := "UPDATE notifications SET read = true WHERE user_id = $1 and id = $2"
	_, err := s.Db.Exec(ctx, query, uid, notificationId)
	if err != nil {
		log.Printf("can not mark all notifications read: %v", err)
		return err
	}

	return nil
}

//Reterive notifications with backware pagination
func (s *Service) Notifications(ctx context.Context, last int, before string) ([]Notification, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return nil, ErrUnAuthorized
	}
	query, args, err := buildQuery(`
	SELECT id, user_id, actors, type, issued_at, read , post_id
	FROM notifications 
	WHERE user_id = @uid
	{{if .before}} 
	AND id < @before
	{{end}}
	ORDER BY id DESC
	{{if .last}}
	LIMIT @last
	{{end}}
	
	
	`, map[string]interface{}{
		"before": before,
		"last":   last,
		"uid":    uid,
	})
	if err != nil {
		return nil, err
	}
	rows, err := s.Db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notifications []Notification
	for rows.Next() {
		var n Notification
		dest := []interface{}{&n.ID, &n.UserId, &n.Actors, &n.Type, &n.Issued_at, &n.Read, &n.PostId}
		if err = rows.Scan(dest...); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("can not scan or iterate through rows: %v", err)
	}
	return notifications, nil
}

//notify follow notification
func (s *Service) NotifyFollow(followerid, followingid int64) {
	ctx := context.Background()
	//Begin transasction
	tx, err := s.Db.Begin(ctx)
	if err != nil {
		log.Printf("can not start the creating post transcation, error: %v", err)
		return
	}
	defer tx.Rollback(ctx)

	query := "SELECT username from users where id = $1"
	var actor string
	if err := tx.QueryRow(ctx, query, followerid).Scan(&actor); err != nil {
		log.Printf("can not get user by id during notifying the followee: %v", err)
		return
	}

	//check if notification already exists
	query = "Select Exists (SELECT 1 from notifications where user_id = $1 and $2::VARCHAR = any(actors) and type = 'follow')"
	var exists bool
	if err := tx.QueryRow(ctx, query, followingid, actor).Scan(&exists); err != nil {
		log.Printf("can not check if notification exists: %v", err)
		return
	}
	if exists {
		return
	}
	var n Notification
	//query to check if there is already a unread notification if not create a new one else update the existing one by appending the actor in that unread notification actors and update issued at returning id,actors,issued_at
	query = "Select id from notifications where user_id = $1 and read = false and type = 'follow'"
	if err = tx.QueryRow(ctx, query, followingid).Scan(&n.ID); err != nil {
		if err == pgx.ErrNoRows {
			query = "INSERT INTO notifications (user_id, actors, type) VALUES ($1, array[$2], 'follow') RETURNING id, actors, issued_at"
			if err = tx.QueryRow(ctx, query, followingid, actor).Scan(&n.ID, &n.Actors, &n.Issued_at); err != nil {
				log.Printf("can not create new notification: %v", err)
				return
			}
		} else {
			log.Printf("can not get unread notification: %v", err)
			return
		}
	} else {
		query = "UPDATE notifications SET actors = array_append(actors, $1), issued_at = now() WHERE id = $2 RETURNING id, actors, issued_at"
		if err = tx.QueryRow(ctx, query, actor, n.ID).Scan(&n.ID, &n.Actors, &n.Issued_at); err != nil {
			log.Printf("can not update notification: %v", err)
			return
		}
	}
	n.UserId = followingid
	n.Type = "follow"
	n.Read = false
	//commit the transaction
	if err := tx.Commit(ctx); err != nil {
		log.Printf("can not commit the creating post transcation, error: %v", err)
		return
	}

	log.Println(litter.Sdump(n))

}

//Comment notification to all the users who commented on the post
func (s *Service) NotifyComment(c Comment) {
	ctx := context.Background()
	actor := c.User.Username

	query := "Insert Into notifications (user_id, actors, type,post_id) Select user_id, array[$1], 'comment',$2 from post_subscriptions where post_id = $2 and user_id != $3 on Conflict (user_id, type,read,post_id) do update set actors = array_prepend($1,array_remove(notifications.actors,$1)),issued_at = now() Returning id,user_id,actors,issued_at"

	rows, err := s.Db.Query(ctx, query, actor, c.PostId, c.UserId)
	if err != nil {
		log.Printf("can not get subscribers: %v", err)
		return
	}
	defer rows.Close()
	var notifications []Notification
	for rows.Next() {
		var n Notification
		dest := []interface{}{&n.ID, &n.UserId, &n.Actors, &n.Issued_at}
		if err = rows.Scan(dest...); err != nil {
			log.Printf("can not scan rows: %v", err)
			return
		}
		n.Type = "comment"
		n.PostId = &c.PostId
		notifications = append(notifications, n)
	}
	if err = rows.Err(); err != nil {
		log.Printf("can not iterate through rows: %v", err)
		return
	}

	log.Println(litter.Sdump(notifications))

}

//Toggle post_subscription
func (s *Service) TogglePostSubscription(ctx context.Context, postid int64) (TogglePostSubscriptionOutput, error) {
	var output TogglePostSubscriptionOutput
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return output, ErrUnAuthorized
	}

	//Begin transasction
	tx, err := s.Db.Begin(ctx)
	if err != nil {
		log.Printf("can not start the creating post transcation, error: %v", err)
		return output, err
	}
	defer tx.Rollback(ctx)

	query := "Select Exists (Select 1 from post_subscriptions where user_id = $1 and post_id = $2)"

	if err := tx.QueryRow(ctx, query, uid, postid).Scan(&output.Subscribed); err != nil {
		return output, err
	}
	if output.Subscribed {
		query = "Delete from post_subscriptions where user_id = $1 and post_id = $2"
		if _, err := s.Db.Exec(ctx, query, uid, postid); err != nil {
			return output, fmt.Errorf("can not delete post subscription: %v", err)
		}
	} else {
		query = "Insert into post_subscriptions (user_id, post_id) values ($1, $2)"
		if _, err := s.Db.Exec(ctx, query, uid, postid); err != nil {
			if isforeignKeyViolation(err) {
				return output, ErrPostNotFound
			} else {
				return output, fmt.Errorf("can not insert post subscription: %v", err)
			}

		}
	}

	//commit the transaction
	if err := tx.Commit(ctx); err != nil {
		log.Printf("can not commit the creating post transcation, error: %v", err)
		return output, err
	}
	output.Subscribed = !output.Subscribed

	return output, nil
}

//notify mention users
func (s *Service) NotifyPostMention(p Post) {
	ctx := context.Background()
	actor := p.User.Username
	mentions := collectMentions(p.Content)
	if len(mentions) == 0 {
		return
	}
	query := "Insert Into notifications (user_id, actors, type,post_id) Select id, array[$1], 'post_mention',$2 from users where users.id != $3 and users.username = any($4) Returning id,user_id,actors,issued_at"

	rows, err := s.Db.Query(ctx, query, actor, p.ID, p.UserId, mentions)
	if err != nil {
		log.Printf("can not insert into post mention notification: %v", err)
		return
	}
	defer rows.Close()
	var notifications []Notification
	for rows.Next() {
		var n Notification
		dest := []interface{}{&n.ID, &n.UserId, &n.Actors, &n.Issued_at}
		if err = rows.Scan(dest...); err != nil {
			log.Printf("can not scan rows: %v", err)
			return
		}
		n.Type = "post_mention"
		n.PostId = &p.ID
		notifications = append(notifications, n)
	}
	if err = rows.Err(); err != nil {
		log.Printf("can not iterate through rows: %v", err)
		return
	}

	log.Println(litter.Sdump(notifications))

}

func (s *Service) NotifyCommentMention(c Comment) {
	ctx := context.Background()
	actor := c.User.Username
	mentions := collectMentions(c.Content)
	if len(mentions) == 0 {
		return
	}
	query := "Insert Into notifications (user_id, actors, type,post_id) Select id, array[$1], 'comment_mention',$2 from users where users.id != $3 and users.username = any($4) on Conflict (user_id, type,read,post_id) do update set actors = array_prepend($1,array_remove(notifications.actors,$1)),issued_at = now()  Returning id,user_id,actors,issued_at"

	rows, err := s.Db.Query(ctx, query, actor, c.PostId, c.UserId, mentions)
	if err != nil {
		log.Printf("can not insert comment mention notification: %v", err)
		return
	}
	defer rows.Close()
	var notifications []Notification
	for rows.Next() {
		var n Notification
		dest := []interface{}{&n.ID, &n.UserId, &n.Actors, &n.Issued_at}
		if err = rows.Scan(dest...); err != nil {
			log.Printf("can not scan rows: %v", err)
			return
		}
		n.Type = "comment_mention"
		n.PostId = &c.PostId
		notifications = append(notifications, n)
	}
	if err = rows.Err(); err != nil {
		log.Printf("can not iterate through rows: %v", err)
		return
	}

	log.Println(litter.Sdump(notifications))

}
