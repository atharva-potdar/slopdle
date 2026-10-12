-- name: GetEnrollmentsByCourse :many
SELECT e.EnrollmentID, e.UserID, e.CourseID, e.RoleID, e.BatchID,
       u.FirstName, u.LastName, u.Email,
       r.RoleName, b.BatchName
FROM Enrollment e
JOIN User u ON e.UserID = u.UserID
JOIN Role r ON e.RoleID = r.RoleID
LEFT JOIN Batch b ON e.BatchID = b.BatchID
WHERE e.CourseID = ?
ORDER BY r.RoleID ASC, u.LastName ASC;

-- name: GetEnrollmentsByUser :many
SELECT e.EnrollmentID, e.UserID, e.CourseID, e.RoleID, e.BatchID,
       c.CourseCode, c.Title AS CourseTitle, c.Description AS CourseDescription,
       r.RoleName, b.BatchName
FROM Enrollment e
JOIN Course c ON e.CourseID = c.CourseID
JOIN Role r ON e.RoleID = r.RoleID
LEFT JOIN Batch b ON e.BatchID = b.BatchID
WHERE e.UserID = ?
ORDER BY c.CourseCode ASC;

-- name: GetUserEnrollmentInCourse :one
SELECT e.EnrollmentID, e.UserID, e.CourseID, e.RoleID, e.BatchID, r.RoleName
FROM Enrollment e
JOIN Role r ON e.RoleID = r.RoleID
WHERE e.UserID = ? AND e.CourseID = ?
LIMIT 1;

-- name: EnrollUser :execresult
INSERT INTO Enrollment (UserID, CourseID, RoleID, BatchID)
VALUES (?, ?, ?, ?);

-- name: UpdateEnrollmentBatch :exec
UPDATE Enrollment
SET BatchID = ?
WHERE EnrollmentID = ?;

-- name: UnenrollUser :exec
DELETE FROM Enrollment
WHERE UserID = ? AND CourseID = ?;

-- name: GetEnrollmentByID :one
SELECT EnrollmentID, UserID, CourseID, RoleID, BatchID
FROM Enrollment
WHERE EnrollmentID = ? LIMIT 1;

-- name: CheckPrerequisiteGrade :one
-- Completion-weighted average over ALL assignments in the required course:
-- each assignment contributes its latest evaluated score, or 0 when the
-- student never submitted or was never graded.
SELECT AVG(COALESCE((
    SELECT e.Score
    FROM Submission s
    JOIN Evaluation e ON e.SubmissionID = s.SubmissionID
    WHERE s.AssignmentID = a.AssignmentID AND s.UserID = ?
    ORDER BY s.SubmissionTime DESC, s.SubmissionID DESC
    LIMIT 1
), 0)) AS AvgScore
FROM Assignment a
WHERE a.CourseID = ?;
