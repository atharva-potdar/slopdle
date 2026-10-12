-- Upload support: per-assignment limits + stored file metadata.
-- ContentData carries the served path (/api/submissions/:id/file) for
-- file submissions; FileName preserves the original upload name.
ALTER TABLE Assignment
  ADD COLUMN MaxUploadMB INT NOT NULL DEFAULT 25,
  ADD COLUMN MaxUploadFiles INT NOT NULL DEFAULT 1;

ALTER TABLE Submission
  ADD COLUMN FileName VARCHAR(255) NULL,
  ADD COLUMN MimeType VARCHAR(100) NULL;
