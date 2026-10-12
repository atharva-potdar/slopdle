-- Notification intents: clicking a notification can deep-link to the
-- item it refers to. Polymorphic target, no FK.
--   announcement -> IntentID = CourseID
--   assignment   -> IntentID = AssignmentID
--   grade        -> IntentID = SubmissionID
--   answer       -> IntentID = QuestionID
-- NULL/NULL = legacy plain notification (mark-read only).

ALTER TABLE Notification
    ADD COLUMN IntentType VARCHAR(30) NULL,
    ADD COLUMN IntentID INT NULL;
