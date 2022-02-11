DROP DATABASE IF EXISTS socially CASCADE;

CREATE DATABASE IF NOT EXISTS socially;

SET DATABASE = socially;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY NOT NULL ,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(255) NOT NULL UNIQUE,
    avatar VARCHAR,
    followers_count INT NOT NULL DEFAULT 0 CHECK (followers_count >= 0),
    followings_count INT NOT NULL DEFAULT 0 CHECK (followings_count >= 0)
);

CREATE TABLE IF NOT EXISTS follows (
    follower_id INT NOT NULL REFERENCES users,
    following_id INT NOT NULL REFERENCES users,
    PRIMARY KEY (follower_id,following_id)
);

CREATE TABLE IF NOT EXISTS posts (
    id SERIAL PRIMARY KEY NOT NULL,
    user_id INT NOT NULL REFERENCES users,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    content VARCHAR NOT NULL,
    likes_count INT NOT NULL DEFAULT 0 CHECK (likes_count >= 0),
    comments_count INT NOT NULL DEFAULT 0 CHECK (likes_count >= 0),
    spoiler VARCHAR,
    nsfw BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS likes (
    user_id INT NOT NULL REFERENCES users,
    post_id INT NOT NULL REFERENCES posts,
    PRIMARY KEY (user_id,post_id)
);

CREATE TABLE IF NOT EXISTS post_subscriptions (
    user_id INT NOT NULL REFERENCES users,
    post_id INT NOT NULL REFERENCES posts,
    PRIMARY KEY (user_id,post_id)
);

CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY NOT NULL,
    user_id INT NOT NULL REFERENCES users,
    post_id INT NOT NULL REFERENCES posts,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    content VARCHAR NOT NULL,
    likes_count INT NOT NULL DEFAULT 0 CHECK (likes_count >= 0)
);

CREATE TABLE IF NOT EXISTS comment_likes (
    user_id INT NOT NULL REFERENCES users,
    comment_id INT NOT NULL REFERENCES comments,
    PRIMARY KEY (user_id,comment_id)
);

CREATE INDEX IF NOT EXISTS posts_created_at_index ON posts (created_at DESC);
CREATE INDEX IF NOT EXISTS comments_created_at_index ON comments (created_at DESC);

CREATE TABLE IF NOT EXISTS timelines (
    id SERIAL PRIMARY KEY NOT NULL,
    user_id INT NOT NULL REFERENCES users,
    post_id INT NOT NULL REFERENCES posts
);

CREATE UNIQUE INDEX IF NOT EXISTS timelines_user_post_index ON timelines (user_id, post_id);

Create TABLE If NOT EXISTS notifications (
    id SERIAL PRIMARY KEY NOT NULL,
    user_id INT NOT NULL REFERENCES users,
    type VARCHAR NOT NULL,
    actors VARCHAR[] NOT NULL,
    post_id INT REFERENCES posts,
    read BOOLEAN NOT NULL DEFAULT FALSE,
    issued_at TIMESTAMP NOT NULL DEFAULT now()
);

Create INDEX If NOT EXISTS notifications_issued_at_index ON notifications (issued_at DESC);
Create UNIQUE INDEX If NOT EXISTS notifications_index ON notifications (user_id, type, read,post_id);





INSERT INTO users (id,email,username,followers_count,followings_count) VALUES (1,'test@test.com','testuser',1,0),(2,'anothertest@test.com','anothertestuser',0,1),(3,'john@test.com','john',0,0),(4,'josh@test.com','josh',0,0);
INSERT INTO follows (follower_id,following_id) VALUES (2,1);
INSERT INTO posts (id,user_id,content,comments_count) VALUES (21,1,'test post by testUser',1),(22,1,'another test post by testUser',1);
INSERT INTO post_subscriptions (user_id,post_id) VALUES (1,21),(1,22);
INSERT INTO timelines (user_id,post_id) VALUES (1,21),(1,22),(2,21),(2,22);
-- INSERT INTO likes (user_id,post_id) VALUES (1,21),(1,22),(2,21),(2,22);
INSERT INTO comments (id,user_id,post_id,content) VALUES (31,1,21,'test comment by testUser'),(32,1,22,'another test comment by testUser');
-- INSERT INTO comment_likes (user_id,comment_id) VALUES (1,31),(1,32),(2,31),(2,32);


