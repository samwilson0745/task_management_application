DELETE FROM task_attachments WHERE storage_path IS NULL;
ALTER TABLE task_attachments ALTER COLUMN storage_path SET NOT NULL;
ALTER TABLE task_attachments DROP COLUMN data;
