CREATE TABLE IF NOT EXISTS post_likes (
    user_id TEXT NOT NULL REFERENCES social_users(id) ON DELETE CASCADE,
    post_id TEXT NOT NULL REFERENCES social_posts(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, post_id)
);

CREATE INDEX IF NOT EXISTS idx_post_likes_post_id
    ON post_likes(post_id);
