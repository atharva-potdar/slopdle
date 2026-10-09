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
