-- Stored upload name for file-based resources (NULL for link rows).
-- FileURL carries the served path (/api/resources/:id/file) for uploads.
ALTER TABLE Resource ADD COLUMN FileName VARCHAR(255) NULL;
