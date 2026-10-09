package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
	"moodleplusplus/internal/middleware"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	UserID       int32    `json:"userId"`
	FirstName    string   `json:"firstName"`
	LastName     string   `json:"lastName"`
	Email        string   `json:"email"`
	Bio          string   `json:"bio"`
	GitHubLink   string   `json:"githubLink"`
	LinkedInLink string   `json:"linkedinLink"`
	Photos       []string `json:"photos"`
}

func (h *AppHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.q.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Passwordhash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token := middleware.Store.Create(user.Userid)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	photos, _ := h.q.GetUserPhotos(r.Context(), sql.NullInt32{Int32: user.Userid, Valid: true})
	var photoURLs []string
	for _, p := range photos {
		if p.Valid {
			photoURLs = append(photoURLs, p.String)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "login successful",
		"token":   token,
		"user": UserResponse{
			UserID:       user.Userid,
			FirstName:    user.Firstname,
			LastName:     user.Lastname,
			Email:        user.Email,
			Bio:          user.Bio.String,
			GitHubLink:   user.Githublink.String,
			LinkedInLink: user.Linkedinlink.String,
			Photos:       photoURLs,
		},
	})
}

func (h *AppHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		middleware.Store.Delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AppHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"user":         nil,
			"enrollments":  []any{},
			"unreadAlerts": 0,
		})
		return
	}

	user, err := h.q.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	photos, _ := h.q.GetUserPhotos(r.Context(), sql.NullInt32{Int32: user.Userid, Valid: true})
	var photoURLs []string
	for _, p := range photos {
		if p.Valid {
			photoURLs = append(photoURLs, p.String)
		}
	}

	enrollments, _ := h.q.GetEnrollmentsByUser(r.Context(), user.Userid)
	unreadCount, _ := h.q.GetUnreadNotificationsCount(r.Context(), user.Userid)

	writeJSON(w, http.StatusOK, map[string]any{
		"user": UserResponse{
			UserID:       user.Userid,
			FirstName:    user.Firstname,
			LastName:     user.Lastname,
			Email:        user.Email,
			Bio:          user.Bio.String,
			GitHubLink:   user.Githublink.String,
			LinkedInLink: user.Linkedinlink.String,
			Photos:       photoURLs,
		},
		"enrollments":  enrollments,
		"unreadAlerts": unreadCount,
	})
}
