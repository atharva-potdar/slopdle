-- name: ListBatchesByCourse :many
SELECT BatchID, BatchName, CourseID
FROM Batch
WHERE CourseID = ?
ORDER BY BatchName ASC;

-- name: GetBatchByID :one
SELECT BatchID, BatchName, CourseID
FROM Batch
WHERE BatchID = ? LIMIT 1;

-- name: CreateBatch :execresult
INSERT INTO Batch (BatchName, CourseID)
VALUES (?, ?);

-- name: DeleteBatch :exec
DELETE FROM Batch WHERE BatchID = ?;
