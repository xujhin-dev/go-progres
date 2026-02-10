-- 用户表添加状态字段和索引
ALTER TABLE users ADD COLUMN IF NOT EXISTS status INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS banned_until TIMESTAMP;

-- 用户表索引
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- 订单表索引
CREATE INDEX IF NOT EXISTS idx_orders_order_no ON orders(order_no);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);

-- 动态表索引
CREATE INDEX IF NOT EXISTS idx_posts_status ON posts(status);
CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);

-- 评论表索引
CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id);
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_root_id ON comments(root_id);
CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_id);

-- 评论表添加 level 字段
ALTER TABLE comments ADD COLUMN IF NOT EXISTS level INTEGER DEFAULT 1;

-- 点赞表索引
CREATE INDEX IF NOT EXISTS idx_likes_user_target ON likes(user_id, target_id, target_type);
CREATE INDEX IF NOT EXISTS idx_likes_target ON likes(target_id, target_type);

-- 用户优惠券表联合唯一索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_coupons_unique ON user_coupons(user_id, coupon_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_coupons_user_id ON user_coupons(user_id);
CREATE INDEX IF NOT EXISTS idx_user_coupons_coupon_id ON user_coupons(coupon_id);
CREATE INDEX IF NOT EXISTS idx_user_coupons_status ON user_coupons(status);

-- 优惠券表索引
CREATE INDEX IF NOT EXISTS idx_coupons_start_time ON coupons(start_time);
CREATE INDEX IF NOT EXISTS idx_coupons_end_time ON coupons(end_time);
