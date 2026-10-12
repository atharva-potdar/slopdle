-- Revert automatic late penalties.
ALTER TABLE Assignment
    DROP COLUMN IF EXISTS LatePenaltyEnabled,
    DROP COLUMN IF EXISTS LatePenaltyPerDay;
