-- name: GetUserNotifications :many
SELECT NotificationID, Message, Type, IsRead, CreatedAt, UserID
FROM Notification
WHERE UserID = ?
ORDER BY CreatedAt DESC
LIMIT 50;

-- name: GetUnreadNotificationsCount :one
SELECT COUNT(*) AS UnreadCount
FROM Notification
WHERE UserID = ? AND IsRead = FALSE;

-- name: CreateNotification :execresult
INSERT INTO Notification (Message, Type, IsRead, CreatedAt, UserID)
VALUES (?, ?, FALSE, NOW(), ?);

-- name: MarkNotificationAsRead :exec
UPDATE Notification
SET IsRead = TRUE
WHERE NotificationID = ? AND UserID = ?;

-- name: MarkAllNotificationsAsRead :exec
UPDATE Notification
SET IsRead = TRUE
WHERE UserID = ?;
