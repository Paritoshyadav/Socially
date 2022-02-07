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
    spoiler VARCHAR,
    nsfw BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS likes (
    user_id INT NOT NULL REFERENCES users,
    post_id INT NOT NULL REFERENCES posts,
    PRIMARY KEY (user_id,post_id)
);

CREATE INDEX IF NOT EXISTS posts_created_at_index ON posts (created_at DESC);

CREATE TABLE IF NOT EXISTS timelines (
    id SERIAL PRIMARY KEY NOT NULL,
    user_id INT NOT NULL REFERENCES users,
    post_id INT NOT NULL REFERENCES posts
);

CREATE UNIQUE INDEX IF NOT EXISTS timelines_user_post_index ON timelines (user_id, post_id);



INSERT INTO users (id,email,username) VALUES (1,'test@test.com','testuser'),(2,'anothertest@test.com','anothertestuser');
INSERT INTO follows (follower_id,following_id) VALUES (2,1);
INSERT INTO posts (id,user_id,content) VALUES (21,1,'test post by testUser'),(22,1,'another test post by testUser');
INSERT INTO timelines (user_id,post_id) VALUES (1,21),(1,22),(2,21),(2,22);

