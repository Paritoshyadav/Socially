SET DATABASE = socially;


-- SELECT *
-- ,following.following_id IS NOT NULL AS following
-- ,followingback.follower_id IS NOT NULL AS followingback
-- FROM follows
-- INNER JOIN users ON users.id = follows.following_id
-- LEFT JOIN follows AS following ON following.follower_id = users.id AND following.following_id = 1
-- LEFT JOIN follows AS followingback ON followingback.following_id =users.id AND followingback.follower_id = 1
-- --WHERE follows.follower_id = (SELECT id From users where username = 'testuser')
-- ORDER BY username ASC;


-- select follower_id,733447560091631600  from follows where following_id = 1;

SELECT posts.id,content, created_at,likes_count,spoiler,nsfw 
	,users.username As username
	,users.avatar As avatar_url
	,posts.user_id = 1 As mine
	,likes.user_id is not null As liked	
	FROM posts 
	Inner join users on users.id = posts.user_id		
	LEFT JOIN likes ON likes.post_id = posts.id AND likes.user_id = 1	
	WHERE posts.id = (SELECT post_id FROM timelines WHERE user_id = 1)
	order by posts.id desc;
	