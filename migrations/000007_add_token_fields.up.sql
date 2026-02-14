-- 为 users 表添加 token 相关字段
ALTER TABLE users 
ADD COLUMN token VARCHAR(500),
ADD COLUMN token_expire_at TIMESTAMP WITH TIME ZONE;

-- 添加索引以提高查询性能
CREATE INDEX idx_users_token_expire_at ON users(token_expire_at);
