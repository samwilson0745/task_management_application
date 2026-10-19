-- Store attachment contents in Postgres so they survive ephemeral container
-- filesystems (e.g. Render's free tier). Rows created before this migration
-- have no data and are treated as unavailable.
ALTER TABLE task_attachments ADD COLUMN data BYTEA;
ALTER TABLE task_attachments ALTER COLUMN storage_path DROP NOT NULL;
