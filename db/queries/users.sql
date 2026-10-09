-- name: GetUserByID :one
SELECT UserID, FirstName, LastName, Email, PasswordHash, Bio, GitHubLink, LinkedInLink
FROM User
WHERE UserID = ? LIMIT 1;

-- name: GetUserByEmail :one
SELECT UserID, FirstName, LastName, Email, PasswordHash, Bio, GitHubLink, LinkedInLink
FROM User
WHERE Email = ? LIMIT 1;

-- name: ListUsers :many
SELECT UserID, FirstName, LastName, Email, Bio, GitHubLink, LinkedInLink
FROM User
ORDER BY UserID ASC;

-- name: CreateUser :execresult
INSERT INTO User (FirstName, LastName, Email, PasswordHash, Bio, GitHubLink, LinkedInLink)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateUserProfile :exec
UPDATE User
SET FirstName = ?, LastName = ?, Bio = ?, GitHubLink = ?, LinkedInLink = ?
WHERE UserID = ?;

-- name: GetUserPhotos :many
SELECT PhotoURL FROM UserPhoto WHERE UserID = ?;

-- name: AddUserPhoto :exec
INSERT INTO UserPhoto (UserID, PhotoURL) VALUES (?, ?);

-- name: DeleteUserPhoto :exec
DELETE FROM UserPhoto WHERE UserID = ? AND PhotoURL = ?;
