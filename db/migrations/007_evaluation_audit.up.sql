-- Internal audit trail for Evaluation changes (INSERT/UPDATE/DELETE).
-- Written by triggers only; no application code touches this table and
-- no endpoint exposes it. Regrades stay allowed, but never silent.
CREATE TABLE IF NOT EXISTS EvaluationAudit (
  AuditID INT NOT NULL AUTO_INCREMENT,
  SubmissionID INT NOT NULL,
  Action VARCHAR(10) NOT NULL,
  OldScore DECIMAL(5,2) NULL,
  NewScore DECIMAL(5,2) NULL,
  OldFeedback TEXT NULL,
  NewFeedback TEXT NULL,
  GraderID INT NULL,
  ChangedAt DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (AuditID),
  KEY SubmissionID (SubmissionID),
  CONSTRAINT chk_EvaluationAudit_Action CHECK (Action IN ('INSERT', 'UPDATE', 'DELETE'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

DROP TRIGGER IF EXISTS trg_evaluation_audit_insert;
DROP TRIGGER IF EXISTS trg_evaluation_audit_update;
DROP TRIGGER IF EXISTS trg_evaluation_audit_delete;

CREATE TRIGGER trg_evaluation_audit_insert
AFTER INSERT ON Evaluation
FOR EACH ROW
INSERT INTO EvaluationAudit (SubmissionID, Action, NewScore, NewFeedback, GraderID)
VALUES (NEW.SubmissionID, 'INSERT', NEW.Score, NEW.Feedback, NEW.UserID);

CREATE TRIGGER trg_evaluation_audit_update
AFTER UPDATE ON Evaluation
FOR EACH ROW
INSERT INTO EvaluationAudit (SubmissionID, Action, OldScore, NewScore, OldFeedback, NewFeedback, GraderID)
VALUES (OLD.SubmissionID, 'UPDATE', OLD.Score, NEW.Score, OLD.Feedback, NEW.Feedback, NEW.UserID);

CREATE TRIGGER trg_evaluation_audit_delete
AFTER DELETE ON Evaluation
FOR EACH ROW
INSERT INTO EvaluationAudit (SubmissionID, Action, OldScore, OldFeedback, GraderID)
VALUES (OLD.SubmissionID, 'DELETE', OLD.Score, OLD.Feedback, OLD.UserID);
