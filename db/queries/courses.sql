-- name: ListCourses :many
SELECT CourseID, CourseCode, Title, Description
FROM Course
ORDER BY CourseCode ASC;

-- name: ListCoursesByUser :many
SELECT c.CourseID, c.CourseCode, c.Title, c.Description
FROM Course c
JOIN Enrollment e ON e.CourseID = c.CourseID
WHERE e.UserID = ?
ORDER BY c.CourseCode ASC;

-- name: GetCourseByID :one
SELECT CourseID, CourseCode, Title, Description
FROM Course
WHERE CourseID = ? LIMIT 1;

-- name: CreateCourse :execresult
INSERT INTO Course (CourseCode, Title, Description)
VALUES (?, ?, ?);

-- name: GetCoursePrerequisites :many
SELECT p.CourseID, p.RequiredCourseID, p.MinPassingGrade,
       c.CourseCode AS RequiredCourseCode, c.Title AS RequiredCourseTitle
FROM Prerequisite p
JOIN Course c ON p.RequiredCourseID = c.CourseID
WHERE p.CourseID = ?;

-- name: AddCoursePrerequisite :exec
INSERT INTO Prerequisite (CourseID, RequiredCourseID, MinPassingGrade)
VALUES (?, ?, ?);

-- name: DeleteCoursePrerequisite :exec
DELETE FROM Prerequisite WHERE CourseID = ? AND RequiredCourseID = ?;
