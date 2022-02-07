package service

import (
	"context"
	"database/sql"
	"fmt"
)

type TimelineItem struct {
	ID     int64 `json:"id"`
	UserId int64 `json:"-"`
	PostId int64 `json:"-"`
	Post   Post  `json:"post"`
}

// Reterive timeline items for a user.
func (s *Service) RetrieveTimelineItems(ctx context.Context, last int, before string) ([]TimelineItem, error) {
	var items []TimelineItem
	var p Post
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return items, ErrUnAuthorized
	}
	query, args, err := buildQuery(`SELECT timelines.id, posts.id,content, created_at,likes_count,spoiler,nsfw 
	,users.username As username
	,users.avatar As avatar_url
	,posts.user_id = @uid As mine
	,likes.user_id is not null As liked	
	FROM timelines
	Inner join posts on posts.id = timelines.post_id 
	Inner join users on users.id = posts.user_id		
	LEFT JOIN likes ON likes.post_id = posts.id AND likes.user_id = @uid	
	WHERE timelines.user_id = @uid
	{{if .before}}
	AND timelines.id < @before
	{{end}}
	order by timelines.id desc
	{{if .last}}
	limit @last
	{{end}}	
	`, map[string]interface{}{
		"last":   last,
		"before": before,
		"uid":    uid,
	})
	if err != nil {
		return items, fmt.Errorf("can not build posts query, error: %v", err)
	}
	var u User
	var avatar sql.NullString
	rows, err := s.Db.Query(ctx, query, args...)
	if err != nil {
		return items, fmt.Errorf("can not get posts, error: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item TimelineItem
		if err = rows.Scan(&item.ID, &p.ID, &p.Content, &p.CreatedAt, &p.LikesCount, &p.SpoilerOf, &p.NSFW, &u.Username, &avatar, &p.IsMe, &p.Liked); err != nil {
			return items, fmt.Errorf("can not scan post, error: %v", err)
		}
		item.UserId = uid
		item.PostId = p.ID
		if avatar.Valid {
			url := s.Origin + "/img/avatars" + avatar.String
			u.AvatarUrl = &url
		}
		p.User = &u
		item.Post = p
		items = append(items, item)
	}

	return items, nil

}
