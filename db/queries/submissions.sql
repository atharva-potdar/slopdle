-- name: CreateSubmission :execresult
INSERT INTO Submission (SubmissionTime, ContentData, AssignmentID, UserID, FileName, MimeType)
VALUES (NOW(), ?, ?, ?, ?, ?);

-- name: DeleteSubmissionByID :exec
DELETE FROM Submission WHERE SubmissionID = ?;

-- name: ListSubmissionFilesByAssignment :many
SELECT SubmissionID, FileName FROM Submission
WHERE AssignmentID = ? AND FileName IS NOT NULL;

-- name: GetSubmissionByID :one
SELECT s.SubmissionID, s.SubmissionTime, s.ContentData, s.AssignmentID, s.UserID, s.FileName, s.MimeType,
       u.FirstName, u.LastName, u.Email,
       a.Title AS AssignmentTitle, a.Deadline, a.CourseID,
       e.Score, e.Feedback, e.UserID AS GradedByUserID
FROM Submission s
JOIN User u ON s.UserID = u.UserID
JOIN Assignment a ON s.AssignmentID = a.AssignmentID
LEFT JOIN Evaluation e ON s.SubmissionID = e.SubmissionID
WHERE s.SubmissionID = ? LIMIT 1;

-- name: GetUserSubmissionForAssignment :one
SELECT s.SubmissionID, s.SubmissionTime, s.ContentData, s.AssignmentID, s.UserID,
       e.Score, e.Feedback, e.UserID AS GradedByUserID
FROM Submission s
LEFT JOIN Evaluation e ON s.SubmissionID = e.SubmissionID
WHERE s.AssignmentID = ? AND s.UserID = ?
LIMIT 1;

-- name: ListSubmissionsByAssignment :many
SELECT s.SubmissionID, s.SubmissionTime, s.ContentData, s.AssignmentID, s.UserID,
       u.FirstName, u.LastName, u.Email,
       e.Score, e.Feedback,
       CASE 
           WHEN s.SubmissionTime > a.Deadline THEN 'LATE'
           ELSE 'ON TIME'
       END AS Timeliness
FROM Submission s
JOIN User u ON s.UserID = u.UserID
JOIN Assignment a ON s.AssignmentID = a.AssignmentID
LEFT JOIN Evaluation e ON s.SubmissionID = e.SubmissionID
WHERE s.AssignmentID = ?
ORDER BY s.SubmissionTime DESC;

-- name: CreateOrUpdateEvaluation :exec
INSERT INTO Evaluation (SubmissionID, Score, Feedback, UserID)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE Score = VALUES(Score), Feedback = VALUES(Feedback), UserID = VALUES(UserID);

-- name: GetSubmissionReport :many
-- Case Study Query 3: Full submission report with timeliness and grader details
SELECT 
    a.AssignmentID,
    a.Title AS AssignmentName,
    CONCAT(Student.FirstName, ' ', Student.LastName) AS StudentName,
    s.UserID AS StudentID,
    s.SubmissionID,
    s.SubmissionTime,
    s.ContentData,
    s.FileName,
    a.Deadline,
    a.LatePenaltyPerDay,
    a.LatePenaltyEnabled,
    CASE 
        WHEN s.SubmissionTime > a.Deadline THEN 'LATE'
        ELSE 'ON TIME'
    END AS Timeliness,
    e.Score,
    CAST(IFNULL(CONCAT(Grader.FirstName, ' ', Grader.LastName), 'Ungraded') AS CHAR(200)) AS GradedBy
FROM Submission s
JOIN Assignment a ON s.AssignmentID = a.AssignmentID
JOIN User Student ON s.UserID = Student.UserID
LEFT JOIN Evaluation e ON s.SubmissionID = e.SubmissionID
LEFT JOIN User Grader ON e.UserID = Grader.UserID
WHERE a.CourseID = ?
ORDER BY (e.Score IS NULL) DESC, s.SubmissionTime DESC;

-- name: GetStudentGradesInCourse :many
-- Case Study Query 1: Assignment scores and feedback for student
SELECT 
    u.FirstName, 
    a.AssignmentID,
    a.Title AS Assignment, 
    e.Score, 
    e.Feedback,
    s.SubmissionTime,
    a.Deadline,
    a.LatePenaltyPerDay,
    a.LatePenaltyEnabled,
    (SELECT GROUP_CONCAT(CONCAT(rc.Title, ' ', ecs.Score, '/', rc.MaxPoints) ORDER BY rc.CriterionID SEPARATOR ' • ')
       FROM EvaluationCriterionScore ecs
       JOIN RubricCriterion rc ON rc.CriterionID = ecs.CriterionID
       WHERE ecs.SubmissionID = s.SubmissionID) AS CriterionBreakdown
FROM User u
JOIN Submission s ON u.UserID = s.UserID
JOIN Assignment a ON s.AssignmentID = a.AssignmentID
JOIN Evaluation e ON s.SubmissionID = e.SubmissionID
WHERE u.UserID = ? AND a.CourseID = ?
ORDER BY s.SubmissionTime DESC, s.SubmissionID DESC;

-- name: ListStudentSubmissionsByCourse :many
SELECT s.SubmissionID, s.SubmissionTime, s.ContentData, s.AssignmentID, s.UserID,
       e.Score, e.Feedback
FROM Submission s
LEFT JOIN Evaluation e ON s.SubmissionID = e.SubmissionID
WHERE s.UserID = ?
  AND s.AssignmentID IN (SELECT AssignmentID FROM Assignment WHERE CourseID = ?)
ORDER BY s.SubmissionTime DESC, s.SubmissionID DESC;

-- name: ListStudentSubmissionsByUser :many
SELECT c.CourseCode,
       c.Title AS CourseTitle,
       a.AssignmentID,
       a.Title AS AssignmentName,
       a.Deadline,
       s.SubmissionID,
       s.SubmissionTime,
       e.Score,
       a.LatePenaltyPerDay,
       a.LatePenaltyEnabled
FROM Submission s
JOIN Assignment a ON s.AssignmentID = a.AssignmentID
JOIN Course c ON a.CourseID = c.CourseID
LEFT JOIN Evaluation e ON s.SubmissionID = e.SubmissionID
WHERE s.UserID = ?
ORDER BY s.SubmissionTime DESC, s.SubmissionID DESC;
