-- 删除索引
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_status;

DROP INDEX IF EXISTS idx_orders_order_no;
DROP INDEX IF EXISTS idx_orders_user_id;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_created_at;

DROP INDEX IF EXISTS idx_posts_status;
DROP INDEX IF EXISTS idx_posts_user_id;
DROP INDEX IF EXISTS idx_posts_created_at;

DROP INDEX IF EXISTS idx_comments_post_id;
DROP INDEX IF EXISTS idx_comments_user_id;
DROP INDEX IF EXISTS idx_comments_root_id;
DROP INDEX IF EXISTS idx_comments_parent_id;

DROP INDEX IF EXISTS idx_likes_user_target;
DROP INDEX IF EXISTS idx_likes_target;

DROP INDEX IF EXISTS idx_user_coupons_unique;
DROP INDEX IF EXISTS idx_user_coupons_user_id;
DROP INDEX IF EXISTS idx_user_coupons_coupon_id;
DROP INDEX IF EXISTS idx_user_coupons_status;

DROP INDEX IF EXISTS idx_coupons_start_time;
DROP INDEX IF EXISTS idx_coupons_end_time;

-- 删除新增字段
ALTER TABLE users DROP COLUMN IF EXISTS status;
ALTER TABLE users DROP COLUMN IF EXISTS banned_until;
ALTER TABLE comments DROP COLUMN IF EXISTS level;
