-- 启用 UUID 扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 删除所有旧表（重置数据库）
DROP TABLE IF EXISTS user_coupons CASCADE;
DROP TABLE IF EXISTS coupons CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS likes CASCADE;
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS post_topics CASCADE;
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS topics CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- 创建 users 表（使用 UUID）
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    username VARCHAR(100),
    password VARCHAR(255),
    email VARCHAR(100),
    mobile VARCHAR(20) UNIQUE,
    nickname VARCHAR(100),
    avatar_url VARCHAR(500),
    role INTEGER DEFAULT 0,
    is_member BOOLEAN DEFAULT FALSE,
    member_expire_at TIMESTAMP WITH TIME ZONE,
    status INTEGER DEFAULT 0,
    banned_until TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_users_mobile ON users(mobile);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- 创建 coupons 表（使用 UUID）
CREATE TABLE coupons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    name VARCHAR(100) NOT NULL,
    total INTEGER NOT NULL,
    stock INTEGER NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_coupons_deleted_at ON coupons(deleted_at);

-- 创建 user_coupons 表（使用 UUID）
CREATE TABLE user_coupons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id UUID NOT NULL,
    coupon_id UUID NOT NULL,
    status INTEGER DEFAULT 1,
    CONSTRAINT fk_user_coupons_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_coupons_coupon FOREIGN KEY (coupon_id) REFERENCES coupons(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_coupons_user_id ON user_coupons(user_id);
CREATE INDEX idx_user_coupons_coupon_id ON user_coupons(coupon_id);
CREATE INDEX idx_user_coupons_deleted_at ON user_coupons(deleted_at);

-- 创建 orders 表（使用 UUID）
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    order_no VARCHAR(100) UNIQUE NOT NULL,
    user_id UUID NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    channel VARCHAR(50),
    subject VARCHAR(200),
    extra_params JSONB,
    paid_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_order_no ON orders(order_no);
CREATE INDEX idx_orders_deleted_at ON orders(deleted_at);

-- 创建 topics 表（使用 UUID）
CREATE TABLE topics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    name VARCHAR(100) UNIQUE NOT NULL
);

CREATE INDEX idx_topics_name ON topics(name);
CREATE INDEX idx_topics_deleted_at ON topics(deleted_at);

-- 创建 posts 表（使用 UUID）
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id UUID NOT NULL,
    content TEXT,
    media_urls JSONB,
    type VARCHAR(50),
    status VARCHAR(50) DEFAULT 'pending',
    CONSTRAINT fk_posts_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_status ON posts(status);
CREATE INDEX idx_posts_deleted_at ON posts(deleted_at);

-- 创建 post_topics 关联表（使用 UUID）
CREATE TABLE post_topics (
    post_id UUID NOT NULL,
    topic_id UUID NOT NULL,
    PRIMARY KEY (post_id, topic_id),
    CONSTRAINT fk_post_topics_post FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_post_topics_topic FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
);

-- 创建 comments 表（使用 UUID）
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    post_id UUID NOT NULL,
    user_id UUID NOT NULL,
    content TEXT,
    parent_id UUID,
    root_id UUID,
    level INTEGER DEFAULT 1,
    CONSTRAINT fk_comments_post FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_comments_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_comments_post_id ON comments(post_id);
CREATE INDEX idx_comments_user_id ON comments(user_id);
CREATE INDEX idx_comments_parent_id ON comments(parent_id);
CREATE INDEX idx_comments_root_id ON comments(root_id);
CREATE INDEX idx_comments_deleted_at ON comments(deleted_at);

-- 创建 likes 表（使用 UUID）
CREATE TABLE likes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    user_id UUID NOT NULL,
    target_id UUID NOT NULL,
    target_type VARCHAR(50),
    CONSTRAINT fk_likes_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_likes_user_id ON likes(user_id);
CREATE INDEX idx_likes_target ON likes(target_id, target_type);
CREATE INDEX idx_likes_deleted_at ON likes(deleted_at);
CREATE UNIQUE INDEX idx_likes_unique ON likes(user_id, target_id, target_type) WHERE deleted_at IS NULL;
