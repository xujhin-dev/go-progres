-- 1. Add new columns
ALTER TABLE users ADD COLUMN IF NOT EXISTS mobile VARCHAR(20) UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS nickname VARCHAR(50);

-- 2. Make password optional (since we use OTP)
ALTER TABLE users ALTER COLUMN password DROP NOT NULL;

-- 3. Make username optional and remove unique constraint (we use mobile as primary identifier)
ALTER TABLE users ALTER COLUMN username DROP NOT NULL;
DROP INDEX IF EXISTS idx_users_username;
-- If there was a unique constraint on username, we should drop it. 
-- However, standard SQL doesn't have a simple "DROP CONSTRAINT IF EXISTS" for unique constraints without knowing the name.
-- Assuming the constraint name is 'users_username_key' or similar if created by GORM/standard SQL.
-- For safety in this migration script, we'll try to drop the standard naming convention constraint.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;

-- 4. Create index for mobile
CREATE INDEX IF NOT EXISTS idx_users_mobile ON users(mobile);
