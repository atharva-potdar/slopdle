-- name: ListCategoriesByCourse :many
SELECT CategoryID, CategoryName, CourseID
FROM ResourceCategory
WHERE CourseID = ?
ORDER BY CategoryName ASC;

-- name: CreateCategory :execresult
INSERT INTO ResourceCategory (CategoryName, CourseID)
VALUES (?, ?);

-- name: GetCategoryByID :one
SELECT CategoryID, CategoryName, CourseID
FROM ResourceCategory
WHERE CategoryID = ? LIMIT 1;

-- name: ListResourcesByCategory :many
SELECT r.ResourceID, r.Title, r.FileURL, r.UploadedAt, r.CategoryID,
       (SELECT GROUP_CONCAT(rb.BatchID ORDER BY rb.BatchID)
          FROM ResourceBatch rb WHERE rb.ResourceID = r.ResourceID) AS BatchIDs,
       (SELECT GROUP_CONCAT(b.BatchName ORDER BY b.BatchName SEPARATOR ', ')
          FROM ResourceBatch rb JOIN Batch b ON b.BatchID = rb.BatchID
          WHERE rb.ResourceID = r.ResourceID) AS BatchNames
FROM Resource r
WHERE r.CategoryID = ?
ORDER BY r.UploadedAt DESC;

-- name: ListResourcesByCategoryForStudent :many
SELECT r.ResourceID, r.Title, r.FileURL, r.UploadedAt, r.CategoryID,
       (SELECT GROUP_CONCAT(rb.BatchID ORDER BY rb.BatchID)
          FROM ResourceBatch rb WHERE rb.ResourceID = r.ResourceID) AS BatchIDs,
       (SELECT GROUP_CONCAT(b.BatchName ORDER BY b.BatchName SEPARATOR ', ')
          FROM ResourceBatch rb JOIN Batch b ON b.BatchID = rb.BatchID
          WHERE rb.ResourceID = r.ResourceID) AS BatchNames
FROM Resource r
JOIN ResourceCategory c ON r.CategoryID = c.CategoryID
WHERE r.CategoryID = ?
  AND (
      NOT EXISTS (SELECT 1 FROM ResourceBatch rb WHERE rb.ResourceID = r.ResourceID)
      OR EXISTS (
          SELECT 1 FROM ResourceBatch rb
          JOIN Enrollment e ON e.BatchID = rb.BatchID
          WHERE rb.ResourceID = r.ResourceID AND e.UserID = ? AND e.CourseID = c.CourseID
      )
  )
ORDER BY r.UploadedAt DESC;

-- name: ListResourcesByCourse :many
SELECT r.ResourceID, r.Title, r.FileURL, r.UploadedAt, r.CategoryID,
       c.CategoryName,
       (SELECT GROUP_CONCAT(rb.BatchID ORDER BY rb.BatchID)
          FROM ResourceBatch rb WHERE rb.ResourceID = r.ResourceID) AS BatchIDs,
       (SELECT GROUP_CONCAT(b.BatchName ORDER BY b.BatchName SEPARATOR ', ')
          FROM ResourceBatch rb JOIN Batch b ON b.BatchID = rb.BatchID
          WHERE rb.ResourceID = r.ResourceID) AS BatchNames
FROM Resource r
JOIN ResourceCategory c ON r.CategoryID = c.CategoryID
WHERE c.CourseID = ?
ORDER BY c.CategoryName ASC, r.UploadedAt DESC;

-- name: ListResourcesByCourseForStudent :many
SELECT r.ResourceID, r.Title, r.FileURL, r.UploadedAt, r.CategoryID,
       c.CategoryName,
       (SELECT GROUP_CONCAT(rb.BatchID ORDER BY rb.BatchID)
          FROM ResourceBatch rb WHERE rb.ResourceID = r.ResourceID) AS BatchIDs,
       (SELECT GROUP_CONCAT(b.BatchName ORDER BY b.BatchName SEPARATOR ', ')
          FROM ResourceBatch rb JOIN Batch b ON b.BatchID = rb.BatchID
          WHERE rb.ResourceID = r.ResourceID) AS BatchNames
FROM Resource r
JOIN ResourceCategory c ON r.CategoryID = c.CategoryID
WHERE c.CourseID = ?
  AND (
      NOT EXISTS (SELECT 1 FROM ResourceBatch rb WHERE rb.ResourceID = r.ResourceID)
      OR EXISTS (
          SELECT 1 FROM ResourceBatch rb
          JOIN Enrollment e ON e.BatchID = rb.BatchID
          WHERE rb.ResourceID = r.ResourceID AND e.UserID = ? AND e.CourseID = c.CourseID
      )
  )
ORDER BY c.CategoryName ASC, r.UploadedAt DESC;

-- name: CreateResource :execresult
INSERT INTO Resource (Title, FileURL, UploadedAt, CategoryID)
VALUES (?, ?, NOW(), ?);

-- name: DeleteResource :exec
DELETE FROM Resource WHERE ResourceID = ?;

-- name: GetResourceByID :one
SELECT r.ResourceID, r.Title, r.FileURL, r.UploadedAt, r.CategoryID, c.CourseID
FROM Resource r
JOIN ResourceCategory c ON r.CategoryID = c.CategoryID
WHERE r.ResourceID = ? LIMIT 1;

-- name: AddResourceBatch :exec
INSERT IGNORE INTO ResourceBatch (ResourceID, BatchID)
VALUES (?, ?);

-- name: DeleteResourceBatches :exec
DELETE FROM ResourceBatch WHERE ResourceID = ?;

-- name: ListResourceBatches :many
SELECT b.BatchID, b.BatchName
FROM ResourceBatch rb
JOIN Batch b ON b.BatchID = rb.BatchID
WHERE rb.ResourceID = ?
ORDER BY b.BatchName ASC;
