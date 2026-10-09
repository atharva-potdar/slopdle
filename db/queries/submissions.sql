-- name: CreateSubmission :execresult
INSERT INTO Submission (SubmissionTime, ContentData, AssignmentID, UserID)
VALUES (NOW(), ?, ?, ?);

-- name: GetSubmissionByID :one
SELECT s.SubmissionID, s.SubmissionTime, s.ContentData, s.AssignmentID, s.UserID,
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
    s.SubmissionID,
    s.SubmissionTime,
    a.Deadline,
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
ORDER BY e.Score DESC;

-- name: GetStudentGradesInCourse :many
-- Case Study Query 1: Assignment scores and feedback for student
SELECT 
    u.FirstName, 
    a.AssignmentID,
    a.Title AS Assignment, 
    e.Score, 
    e.Feedback
FROM User u
JOIN Submission s ON u.UserID = s.UserID
JOIN Assignment a ON s.AssignmentID = a.AssignmentID
JOIN Evaluation e ON s.SubmissionID = e.SubmissionID
WHERE u.UserID = ? AND a.CourseID = ?
ORDER BY s.SubmissionTime DESC, s.SubmissionID DESC;
