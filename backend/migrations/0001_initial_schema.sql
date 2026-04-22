CREATE TABLE IF NOT EXISTS social_users (
    id TEXT PRIMARY KEY,
    handle TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    bio TEXT NOT NULL DEFAULT '',
    instance TEXT NOT NULL,
    wallet TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    fields_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    featured_tags_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_bot BOOLEAN NOT NULL DEFAULT FALSE,
    followers_count INTEGER NOT NULL DEFAULT 0,
    following_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS auth_accounts (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    wallet TEXT NOT NULL DEFAULT '' UNIQUE,
    user_id TEXT NOT NULL REFERENCES social_users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS media_assets (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES social_users(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    url TEXT NOT NULL DEFAULT '',
    storage_uri TEXT NOT NULL DEFAULT '',
    cid TEXT NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'uploaded',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS social_posts (
    id TEXT PRIMARY KEY,
    author_id TEXT NOT NULL REFERENCES social_users(id) ON DELETE RESTRICT,
    kind TEXT NOT NULL,
    content TEXT NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'public',
    storage_uri TEXT NOT NULL DEFAULT '',
    attestation_uri TEXT NOT NULL DEFAULT '',
    chain_id TEXT NOT NULL DEFAULT '',
    tx_hash TEXT NOT NULL DEFAULT '',
    contract_address TEXT NOT NULL DEFAULT '',
    explorer_url TEXT NOT NULL DEFAULT '',
    tags_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    parent_post_id TEXT REFERENCES social_posts(id) ON DELETE SET NULL,
    root_post_id TEXT REFERENCES social_posts(id) ON DELETE SET NULL,
    reply_depth INTEGER NOT NULL DEFAULT 0,
    replies_count INTEGER NOT NULL DEFAULT 0,
    boosts_count INTEGER NOT NULL DEFAULT 0,
    likes_count INTEGER NOT NULL DEFAULT 0,
    type TEXT NOT NULL DEFAULT 'post',
    interaction TEXT NOT NULL DEFAULT 'anyone',
    poll_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS post_media_links (
    post_id TEXT NOT NULL REFERENCES social_posts(id) ON DELETE CASCADE,
    media_id TEXT NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, media_id)
);

CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    initiator_id TEXT REFERENCES social_users(id) ON DELETE SET NULL,
    encrypted BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS conversation_participants (
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES social_users(id) ON DELETE CASCADE,
    PRIMARY KEY (conversation_id, user_id)
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id TEXT NOT NULL REFERENCES social_users(id) ON DELETE RESTRICT,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS federation_instances (
    name TEXT PRIMARY KEY,
    focus TEXT NOT NULL,
    members TEXT NOT NULL,
    latency TEXT NOT NULL,
    status TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_follows (
    follower_id TEXT NOT NULL REFERENCES social_users(id) ON DELETE CASCADE,
    followee_id TEXT NOT NULL REFERENCES social_users(id) ON DELETE CASCADE,
    PRIMARY KEY (follower_id, followee_id),
    CHECK (follower_id <> followee_id)
);

CREATE INDEX IF NOT EXISTS idx_social_posts_author_created_at
    ON social_posts(author_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_social_posts_root_created_at
    ON social_posts(root_post_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_media_assets_owner_created_at
    ON media_assets(owner_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_created_at
    ON chat_messages(conversation_id, created_at DESC);
