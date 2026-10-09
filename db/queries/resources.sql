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
SELECT ResourceID, Title, FileURL, UploadedAt, CategoryID
FROM Resource
WHERE CategoryID = ?
ORDER BY UploadedAt DESC;

-- name: ListResourcesByCourse :many
SELECT r.ResourceID, r.Title, r.FileURL, r.UploadedAt, r.CategoryID,
       c.CategoryName
FROM Resource r
JOIN ResourceCategory c ON r.CategoryID = c.CategoryID
WHERE c.CourseID = ?
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
