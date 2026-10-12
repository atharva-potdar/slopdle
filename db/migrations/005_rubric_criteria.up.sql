-- Departmental rubric criteria and per-criterion evaluation scores.
-- A rubric's MaxPoints total defines its scale; criterion sums are
-- normalized to /100 at grading time, so totals other than 100 work.

CREATE TABLE IF NOT EXISTS RubricCriterion (
    CriterionID INT AUTO_INCREMENT PRIMARY KEY,
    RubricID INT NOT NULL,
    Title VARCHAR(255) NOT NULL,
    MaxPoints DECIMAL(6,2) NOT NULL,
    CONSTRAINT chk_Criterion_Points CHECK (MaxPoints > 0),
    FOREIGN KEY (RubricID) REFERENCES Rubric(RubricID) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS EvaluationCriterionScore (
    SubmissionID INT NOT NULL,
    CriterionID INT NOT NULL,
    Score DECIMAL(6,2) NOT NULL,
    Feedback TEXT,
    PRIMARY KEY (SubmissionID, CriterionID),
    CONSTRAINT chk_CriterionScore_NonNeg CHECK (Score >= 0),
    FOREIGN KEY (SubmissionID) REFERENCES Submission(SubmissionID) ON DELETE CASCADE,
    FOREIGN KEY (CriterionID) REFERENCES RubricCriterion(CriterionID) ON DELETE CASCADE
);
