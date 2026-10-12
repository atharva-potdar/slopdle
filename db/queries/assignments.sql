-- name: ListAssignmentsByCourse :many
SELECT a.AssignmentID, a.Title, a.Description, a.Deadline, a.FormatRequired, a.CourseID, a.RubricID,
       r.Title AS RubricTitle, r.Department AS RubricDepartment,
       (SELECT GROUP_CONCAT(ab.BatchID ORDER BY ab.BatchID)
          FROM AssignmentBatch ab WHERE ab.AssignmentID = a.AssignmentID) AS BatchIDs,
       (SELECT GROUP_CONCAT(b.BatchName ORDER BY b.BatchName SEPARATOR ', ')
          FROM AssignmentBatch ab JOIN Batch b ON b.BatchID = ab.BatchID
          WHERE ab.AssignmentID = a.AssignmentID) AS BatchNames
FROM Assignment a
LEFT JOIN Rubric r ON a.RubricID = r.RubricID
WHERE a.CourseID = ?
ORDER BY a.Deadline ASC;

-- name: ListAssignmentsByCourseForStudent :many
-- Course-wide assignments (no batch rows) plus assignments targeted at the
-- student's own batch.
SELECT a.AssignmentID, a.Title, a.Description, a.Deadline, a.FormatRequired, a.CourseID, a.RubricID,
       r.Title AS RubricTitle, r.Department AS RubricDepartment,
       (SELECT GROUP_CONCAT(ab.BatchID ORDER BY ab.BatchID)
          FROM AssignmentBatch ab WHERE ab.AssignmentID = a.AssignmentID) AS BatchIDs,
       (SELECT GROUP_CONCAT(b.BatchName ORDER BY b.BatchName SEPARATOR ', ')
          FROM AssignmentBatch ab JOIN Batch b ON b.BatchID = ab.BatchID
          WHERE ab.AssignmentID = a.AssignmentID) AS BatchNames
FROM Assignment a
LEFT JOIN Rubric r ON a.RubricID = r.RubricID
WHERE a.CourseID = ?
  AND (
      NOT EXISTS (SELECT 1 FROM AssignmentBatch ab WHERE ab.AssignmentID = a.AssignmentID)
      OR EXISTS (
          SELECT 1 FROM AssignmentBatch ab
          JOIN Enrollment e ON e.BatchID = ab.BatchID
          WHERE ab.AssignmentID = a.AssignmentID AND e.UserID = ? AND e.CourseID = a.CourseID
      )
  )
ORDER BY a.Deadline ASC;

-- name: GetAssignmentByID :one
SELECT a.AssignmentID, a.Title, a.Description, a.Deadline, a.FormatRequired, a.CourseID, a.RubricID,
       r.Title AS RubricTitle, r.Department AS RubricDepartment, r.Description AS RubricDescription,
       c.CourseCode, c.Title AS CourseTitle,
       (SELECT GROUP_CONCAT(ab.BatchID ORDER BY ab.BatchID)
          FROM AssignmentBatch ab WHERE ab.AssignmentID = a.AssignmentID) AS BatchIDs,
       (SELECT GROUP_CONCAT(b.BatchName ORDER BY b.BatchName SEPARATOR ', ')
          FROM AssignmentBatch ab JOIN Batch b ON b.BatchID = ab.BatchID
          WHERE ab.AssignmentID = a.AssignmentID) AS BatchNames
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

-- name: AddAssignmentBatch :exec
INSERT IGNORE INTO AssignmentBatch (AssignmentID, BatchID)
VALUES (?, ?);

-- name: DeleteAssignmentBatches :exec
DELETE FROM AssignmentBatch WHERE AssignmentID = ?;

-- name: ListAssignmentBatches :many
SELECT b.BatchID, b.BatchName
FROM AssignmentBatch ab
JOIN Batch b ON b.BatchID = ab.BatchID
WHERE ab.AssignmentID = ?
ORDER BY b.BatchName ASC;

-- name: CountAssignmentVisibleToStudent :one
-- Returns 1 when the assignment is course-wide or targeted at the student's batch.
SELECT COUNT(*)
FROM Assignment a
WHERE a.AssignmentID = ?
  AND a.CourseID = ?
  AND (
      NOT EXISTS (SELECT 1 FROM AssignmentBatch ab WHERE ab.AssignmentID = a.AssignmentID)
      OR EXISTS (
          SELECT 1 FROM AssignmentBatch ab
          JOIN Enrollment e ON e.BatchID = ab.BatchID
          WHERE ab.AssignmentID = a.AssignmentID AND e.UserID = ? AND e.CourseID = a.CourseID
      )
  );

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
  AND (
      NOT EXISTS (SELECT 1 FROM AssignmentBatch ab WHERE ab.AssignmentID = a.AssignmentID)
      OR e.BatchID IN (SELECT ab.BatchID FROM AssignmentBatch ab WHERE ab.AssignmentID = a.AssignmentID)
  )
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
  AND (
      NOT EXISTS (SELECT 1 FROM AssignmentBatch ab WHERE ab.AssignmentID = a.AssignmentID)
      OR e.BatchID IN (SELECT ab.BatchID FROM AssignmentBatch ab WHERE ab.AssignmentID = a.AssignmentID)
  )
ORDER BY a.Deadline ASC, u.LastName ASC;

-- name: GetPendingAssignmentsForStudent :many
SELECT 
    c.CourseID,
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
  AND (
      NOT EXISTS (SELECT 1 FROM AssignmentBatch ab WHERE ab.AssignmentID = a.AssignmentID)
      OR e.BatchID IN (SELECT ab.BatchID FROM AssignmentBatch ab WHERE ab.AssignmentID = a.AssignmentID)
  )
ORDER BY a.Deadline ASC;
