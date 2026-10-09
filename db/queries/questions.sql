-- name: ListQuestionsByCourse :many
SELECT q.QuestionID, q.Title, q.Body, q.CreatedAt, q.CourseID, q.UserID,
       u.FirstName, u.LastName,
       COUNT(a.AnswerID) AS AnswerCount,
       MAX(CASE WHEN a.IsOfficialAnswer = TRUE THEN 1 ELSE 0 END) AS HasOfficialAnswer
FROM Question q
JOIN User u ON q.UserID = u.UserID
LEFT JOIN Answer a ON q.QuestionID = a.QuestionID
WHERE q.CourseID = ?
GROUP BY q.QuestionID, q.Title, q.Body, q.CreatedAt, q.CourseID, q.UserID, u.FirstName, u.LastName
ORDER BY q.CreatedAt DESC;

-- name: GetQuestionByID :one
SELECT q.QuestionID, q.Title, q.Body, q.CreatedAt, q.CourseID, q.UserID,
       u.FirstName, u.LastName, u.Email
FROM Question q
JOIN User u ON q.UserID = u.UserID
WHERE q.QuestionID = ? LIMIT 1;

-- name: CreateQuestion :execresult
INSERT INTO Question (Title, Body, CourseID, UserID)
VALUES (?, ?, ?, ?);

-- name: ListAnswersByQuestion :many
SELECT a.AnswerID, a.Body, a.Upvotes, a.IsOfficialAnswer, a.CreatedAt, a.QuestionID, a.UserID,
       u.FirstName, u.LastName, u.Email,
       r.RoleName AS UserRole,
       EXISTS(SELECT 1 FROM AnswerUpvote au WHERE au.AnswerID = a.AnswerID AND au.UserID = ?) AS HasUpvoted
FROM Answer a
JOIN User u ON a.UserID = u.UserID
JOIN Question q ON a.QuestionID = q.QuestionID
LEFT JOIN Enrollment e ON (e.UserID = a.UserID AND e.CourseID = q.CourseID)
LEFT JOIN Role r ON e.RoleID = r.RoleID
WHERE a.QuestionID = ?
ORDER BY a.IsOfficialAnswer DESC, a.Upvotes DESC, a.CreatedAt ASC;

-- name: GetAnswerByID :one
SELECT a.AnswerID, a.Body, a.Upvotes, a.IsOfficialAnswer, a.QuestionID, a.UserID,
       q.CourseID
FROM Answer a
JOIN Question q ON a.QuestionID = q.QuestionID
WHERE a.AnswerID = ? LIMIT 1;

-- name: CreateAnswer :execresult
INSERT INTO Answer (Body, QuestionID, UserID)
VALUES (?, ?, ?);

-- name: HasAnswerUpvote :one
SELECT COUNT(*) FROM AnswerUpvote
WHERE AnswerID = ? AND UserID = ?;

-- name: AddAnswerUpvote :exec
INSERT INTO AnswerUpvote (AnswerID, UserID)
VALUES (?, ?);

-- name: RemoveAnswerUpvote :exec
DELETE FROM AnswerUpvote
WHERE AnswerID = ? AND UserID = ?;

-- name: IncrementAnswerUpvotes :exec
UPDATE Answer SET Upvotes = Upvotes + 1
WHERE AnswerID = ?;

-- name: DecrementAnswerUpvotes :exec
UPDATE Answer SET Upvotes = GREATEST(Upvotes - 1, 0)
WHERE AnswerID = ?;

-- name: MarkOfficialAnswer :exec
UPDATE Answer
SET IsOfficialAnswer = ?
WHERE AnswerID = ?;
