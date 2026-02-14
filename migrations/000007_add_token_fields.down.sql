-- 删除 token 相关字段和索引
DROP INDEX IF EXISTS idx_users_token_expire_at;
ALTER TABLE users 
DROP COLUMN IF EXISTS token,
DROP COLUMN IF EXISTS token_expire_at;
