-- Drop triggers
DROP TRIGGER IF EXISTS update_functions_updated_at ON functions;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS function_permissions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS invocations;
DROP TABLE IF EXISTS functions;

-- Drop extension
DROP EXTENSION IF EXISTS "uuid-ossp";
