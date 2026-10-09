package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"moodleplusplus/internal/db"
)

type UpdateUserRequest struct {
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	Bio          string `json:"bio"`
	GitHubLink   string `json:"githubLink"`
	LinkedInLink string `json:"linkedinLink"`
}

type PhotoRequest struct {
	PhotoURL string `json:"photoUrl"`
}

func (h *AppHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if !h.requireAnyStaff(w, r) {
		return
	}

	users, err := h.q.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]map[string]any, 0, len(users))
	for _, u := range users {
		items = append(items, map[string]any{
			"userId":       u.Userid,
			"firstName":    u.Firstname,
			"lastName":     u.Lastname,
			"email":        u.Email,
			"bio":          u.Bio.String,
			"githubLink":   u.Githublink.String,
			"linkedinLink": u.Linkedinlink.String,
		})
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *AppHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := h.q.GetUserByID(r.Context(), id)
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

	writeJSON(w, http.StatusOK, map[string]any{
		"userId":       user.Userid,
		"firstName":    user.Firstname,
		"lastName":     user.Lastname,
		"email":        user.Email,
		"bio":          user.Bio.String,
		"githubLink":   user.Githublink.String,
		"linkedinLink": user.Linkedinlink.String,
		"photos":       photoURLs,
	})
}

func (h *AppHandler) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.q.UpdateUserProfile(r.Context(), db.UpdateUserProfileParams{
		Firstname:    req.FirstName,
		Lastname:     req.LastName,
		Bio:          sql.NullString{String: req.Bio, Valid: req.Bio != ""},
		Githublink:   sql.NullString{String: req.GitHubLink, Valid: req.GitHubLink != ""},
		Linkedinlink: sql.NullString{String: req.LinkedInLink, Valid: req.LinkedInLink != ""},
		Userid:       id,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "profile updated"})
}

func (h *AppHandler) AddUserPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req PhotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PhotoURL == "" {
		writeError(w, http.StatusBadRequest, "valid photoUrl required")
		return
	}

	err = h.q.AddUserPhoto(r.Context(), db.AddUserPhotoParams{
		Userid:   sql.NullInt32{Int32: id, Valid: true},
		Photourl: sql.NullString{String: req.PhotoURL, Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "photo added"})
}

func (h *AppHandler) DeleteUserPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req PhotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PhotoURL == "" {
		writeError(w, http.StatusBadRequest, "valid photoUrl required")
		return
	}

	err = h.q.DeleteUserPhoto(r.Context(), db.DeleteUserPhotoParams{
		Userid:   sql.NullInt32{Int32: id, Valid: true},
		Photourl: sql.NullString{String: req.PhotoURL, Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "photo removed"})
}
