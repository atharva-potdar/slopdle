-- name: ListAssignmentsByCourse :many
SELECT a.AssignmentID, a.Title, a.Description, a.Deadline, a.FormatRequired, a.CourseID, a.RubricID,
       r.Title AS RubricTitle, r.Department AS RubricDepartment
FROM Assignment a
LEFT JOIN Rubric r ON a.RubricID = r.RubricID
WHERE a.CourseID = ?
ORDER BY a.Deadline ASC;

-- name: GetAssignmentByID :one
SELECT a.AssignmentID, a.Title, a.Description, a.Deadline, a.FormatRequired, a.CourseID, a.RubricID,
       r.Title AS RubricTitle, r.Department AS RubricDepartment, r.Description AS RubricDescription,
       c.CourseCode, c.Title AS CourseTitle
FROM Assignment a
JOIN Course c ON a.CourseID = c.CourseID
LEFT JOIN Rubric r ON a.RubricID = r.RubricID
WHERE a.AssignmentID = ? LIMIT 1;

-- name: CreateAssignment :execresult
INSERT INTO Assignment (Title, Description, Deadline, FormatRequired, CourseID, RubricID)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateAssignment :exec
UPDATE Assignment
SET Title = ?, Description = ?, Deadline = ?, FormatRequired = ?, RubricID = ?
WHERE AssignmentID = ?;

-- name: DeleteAssignment :exec
DELETE FROM Assignment WHERE AssignmentID = ?;

-- name: GetPendingStudentsForAssignment :many
-- Case Study Query 2: Students enrolled in course who have not submitted for this assignment
SELECT 
    c.CourseCode,
    a.Title AS AssignmentName,
    a.Deadline,
    u.UserID,
    u.FirstName,
    u.LastName,
    u.Email
FROM Assignment a
JOIN Course c ON a.CourseID = c.CourseID
JOIN Enrollment e ON c.CourseID = e.CourseID
JOIN Role r ON e.RoleID = r.RoleID 
JOIN User u ON e.UserID = u.UserID
LEFT JOIN Submission s ON a.AssignmentID = s.AssignmentID AND u.UserID = s.UserID
WHERE a.AssignmentID = ?
  AND r.RoleName = 'Student'
  AND s.SubmissionID IS NULL
ORDER BY u.LastName ASC;

-- name: GetPendingStudentsForCourse :many
SELECT 
    c.CourseCode,
    a.Title AS AssignmentName,
    a.Deadline,
    u.UserID,
    u.FirstName,
    u.LastName,
    u.Email
FROM Assignment a
JOIN Course c ON a.CourseID = c.CourseID
JOIN Enrollment e ON c.CourseID = e.CourseID
JOIN Role r ON e.RoleID = r.RoleID 
JOIN User u ON e.UserID = u.UserID
LEFT JOIN Submission s ON a.AssignmentID = s.AssignmentID AND u.UserID = s.UserID
WHERE c.CourseID = ?
  AND r.RoleName = 'Student'
  AND s.SubmissionID IS NULL
ORDER BY a.Deadline ASC, u.LastName ASC;

-- name: GetPendingAssignmentsForStudent :many
SELECT 
    c.CourseCode,
    c.Title AS CourseTitle,
    a.AssignmentID,
    a.Title AS AssignmentName,
    a.Deadline,
    a.FormatRequired
FROM Assignment a
JOIN Course c ON a.CourseID = c.CourseID
JOIN Enrollment e ON c.CourseID = e.CourseID
LEFT JOIN Submission s ON a.AssignmentID = s.AssignmentID AND e.UserID = s.UserID
WHERE e.UserID = ?
  AND s.SubmissionID IS NULL
ORDER BY a.Deadline ASC;
