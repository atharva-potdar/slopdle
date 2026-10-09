-- name: ListSessionsByBatch :many
SELECT s.SessionID, s.StartTime, s.EndTime, s.RoomLocation, s.BatchID,
       b.BatchName, b.CourseID
FROM ClassSession s
JOIN Batch b ON s.BatchID = b.BatchID
WHERE s.BatchID = ?
ORDER BY s.StartTime ASC;

-- name: ListSessionsByCourse :many
SELECT s.SessionID, s.StartTime, s.EndTime, s.RoomLocation, s.BatchID,
       b.BatchName, b.CourseID
FROM ClassSession s
JOIN Batch b ON s.BatchID = b.BatchID
WHERE b.CourseID = ?
ORDER BY s.StartTime ASC;

-- name: CreateClassSession :execresult
INSERT INTO ClassSession (StartTime, EndTime, RoomLocation, BatchID)
VALUES (?, ?, ?, ?);

-- name: GetSessionByID :one
SELECT s.SessionID, s.StartTime, s.EndTime, s.RoomLocation, s.BatchID,
       b.BatchName, b.CourseID
FROM ClassSession s
JOIN Batch b ON s.BatchID = b.BatchID
WHERE s.SessionID = ? LIMIT 1;

-- name: RecordAttendance :exec
INSERT INTO Attendance (SessionID, UserID, Status)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE Status = VALUES(Status);

-- name: GetSessionAttendance :many
SELECT a.SessionID, a.UserID, a.Status,
       u.FirstName, u.LastName, u.Email
FROM Attendance a
JOIN User u ON a.UserID = u.UserID
WHERE a.SessionID = ?
ORDER BY u.LastName ASC;

-- name: GetUserAttendanceInCourse :many
SELECT s.SessionID, s.StartTime, s.EndTime, s.RoomLocation, b.BatchName,
       IFNULL(att.Status, 'Unmarked') AS AttendanceStatus
FROM ClassSession s
JOIN Batch b ON s.BatchID = b.BatchID
JOIN Enrollment e ON (e.CourseID = b.CourseID AND (e.BatchID IS NULL OR e.BatchID = b.BatchID))
LEFT JOIN Attendance att ON (att.SessionID = s.SessionID AND att.UserID = e.UserID)
WHERE b.CourseID = ? AND e.UserID = ?
ORDER BY s.StartTime ASC;

-- name: GetUserAttendanceSummary :one
SELECT 
    COUNT(s.SessionID) AS TotalSessions,
    SUM(CASE WHEN att.Status = 'Present' THEN 1 ELSE 0 END) AS PresentCount,
    SUM(CASE WHEN att.Status = 'Absent' THEN 1 ELSE 0 END) AS AbsentCount,
    SUM(CASE WHEN att.Status = 'Excused' THEN 1 ELSE 0 END) AS ExcusedCount
FROM ClassSession s
JOIN Batch b ON s.BatchID = b.BatchID
JOIN Enrollment e ON (e.CourseID = b.CourseID AND (e.BatchID IS NULL OR e.BatchID = b.BatchID))
LEFT JOIN Attendance att ON (att.SessionID = s.SessionID AND att.UserID = e.UserID)
WHERE b.CourseID = ? AND e.UserID = ?;
