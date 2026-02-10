-- User enhancements
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_member BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS member_expire_at TIMESTAMP WITH TIME ZONE;

-- Moments Module

-- 1. Topics
CREATE TABLE IF NOT EXISTS topics (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Posts
CREATE TABLE IF NOT EXISTS posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    content TEXT,
    media_urls JSONB, -- Store array of URLs: ["http://...", "http://..."]
    type VARCHAR(20), -- 'text', 'image', 'video'
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'approved', 'rejected'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_posts_user FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id);
CREATE INDEX IF NOT EXISTS idx_posts_status ON posts(status);
CREATE INDEX IF NOT EXISTS idx_posts_deleted_at ON posts(deleted_at);

-- 3. Post Topics (Many-to-Many)
CREATE TABLE IF NOT EXISTS post_topics (
    post_id INTEGER,
    topic_id INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (post_id, topic_id),
    CONSTRAINT fk_post_topics_post FOREIGN KEY (post_id) REFERENCES posts(id),
    CONSTRAINT fk_post_topics_topic FOREIGN KEY (topic_id) REFERENCES topics(id)
);

-- 4. Comments
CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    parent_id INTEGER DEFAULT 0, -- 0 for root comment, otherwise parent comment id
    root_id INTEGER DEFAULT 0, -- For optimizing thread fetching
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_comments_post FOREIGN KEY (post_id) REFERENCES posts(id),
    CONSTRAINT fk_comments_user FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id);

-- 5. Likes
CREATE TABLE IF NOT EXISTS likes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    target_id INTEGER NOT NULL, -- post_id or comment_id
    target_type VARCHAR(20) NOT NULL, -- 'post' or 'comment'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, target_id, target_type),
    CONSTRAINT fk_likes_user FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_likes_target ON likes(target_id, target_type);
