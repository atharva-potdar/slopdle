-- Automatic late penalties per assignment.
-- LatePenaltyEnabled = FALSE (or 0.00/day) disables deductions.

ALTER TABLE Assignment
    ADD COLUMN LatePenaltyPerDay DECIMAL(5,2) NOT NULL DEFAULT 10.00,
    ADD COLUMN LatePenaltyEnabled BOOLEAN NOT NULL DEFAULT TRUE;
