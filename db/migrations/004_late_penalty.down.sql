-- Revert automatic late penalties.
ALTER TABLE Assignment
    DROP COLUMN LatePenaltyEnabled,
    DROP COLUMN LatePenaltyPerDay;
