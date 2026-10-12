-- Revert notification intents.
ALTER TABLE Notification
    DROP COLUMN IF EXISTS IntentID,
    DROP COLUMN IF EXISTS IntentType;
