-- name: ListRubrics :many
SELECT RubricID, Title, Department, Description
FROM Rubric
ORDER BY Title ASC;

-- name: GetRubricByID :one
SELECT RubricID, Title, Department, Description
FROM Rubric
WHERE RubricID = ? LIMIT 1;

-- name: CreateRubric :execresult
INSERT INTO Rubric (Title, Department, Description)
VALUES (?, ?, ?);

-- name: UpdateRubric :exec
UPDATE Rubric
SET Title = ?, Department = ?, Description = ?
WHERE RubricID = ?;

-- name: ListRubricCriteria :many
SELECT CriterionID, RubricID, Title, MaxPoints
FROM RubricCriterion
WHERE RubricID = ?
ORDER BY CriterionID ASC;

-- name: GetCriterionByID :one
SELECT CriterionID, RubricID, Title, MaxPoints
FROM RubricCriterion
WHERE CriterionID = ? LIMIT 1;

-- name: CreateCriterion :execresult
INSERT INTO RubricCriterion (RubricID, Title, MaxPoints)
VALUES (?, ?, ?);

-- name: UpdateCriterion :exec
UPDATE RubricCriterion
SET Title = ?, MaxPoints = ?
WHERE CriterionID = ?;

-- name: DeleteCriterion :exec
DELETE FROM RubricCriterion WHERE CriterionID = ?;

-- name: UpsertCriterionScore :exec
INSERT INTO EvaluationCriterionScore (SubmissionID, CriterionID, Score, Feedback)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE Score = VALUES(Score), Feedback = VALUES(Feedback);

-- name: ListCriterionScoresBySubmission :many
SELECT ecs.SubmissionID, ecs.CriterionID, ecs.Score, ecs.Feedback,
       rc.Title AS CriterionTitle, rc.MaxPoints
FROM EvaluationCriterionScore ecs
JOIN RubricCriterion rc ON rc.CriterionID = ecs.CriterionID
WHERE ecs.SubmissionID = ?
ORDER BY ecs.CriterionID ASC;
