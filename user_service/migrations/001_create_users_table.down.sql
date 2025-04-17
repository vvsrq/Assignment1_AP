DROP TRIGGER IF EXISTS set_users_timestamp ON users;
DROP FUNCTION IF EXISTS trigger_set_timestamp(); -- Drop only if it's specific to this table
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;