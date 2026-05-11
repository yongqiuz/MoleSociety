ALTER TABLE social_posts
    ADD COLUMN IF NOT EXISTS bookmark_count INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS post_bookmarks (
    user_id TEXT NOT NULL REFERENCES social_users(id) ON DELETE CASCADE,
    post_id TEXT NOT NULL REFERENCES social_posts(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, post_id)
);

CREATE INDEX IF NOT EXISTS idx_post_bookmarks_post_id
    ON post_bookmarks(post_id);
