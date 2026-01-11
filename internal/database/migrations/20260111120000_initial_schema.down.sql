-- Rollback initial schema
-- Drops all tables and indexes created in up migration

-- Drop indexes first
DROP INDEX IF EXISTS idx_configs_name;
DROP INDEX IF EXISTS idx_executions_started_at;
DROP INDEX IF EXISTS idx_executions_deployment_id;
DROP INDEX IF EXISTS idx_deployments_config_id;
DROP INDEX IF EXISTS idx_deployments_status;
DROP INDEX IF EXISTS idx_deployments_name;

-- Drop tables in reverse order of creation (due to foreign keys)
DROP TABLE IF EXISTS executions;
DROP TABLE IF EXISTS deployments;
DROP TABLE IF EXISTS configs;
