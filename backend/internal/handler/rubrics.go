package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"moodleplusplus/internal/db"
)

type CreateRubricRequest struct {
	Title       string `json:"title"`
	Department  string `json:"department"`
	Description string `json:"description"`
}

func (h *AppHandler) ListRubrics(w http.ResponseWriter, r *http.Request) {
	rubrics, err := h.q.ListRubrics(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rubrics)
}

func (h *AppHandler) GetRubric(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rubric id")
		return
	}

	rubric, err := h.q.GetRubricByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "rubric not found")
		return
	}
	writeJSON(w, http.StatusOK, rubric)
}

func (h *AppHandler) CreateRubric(w http.ResponseWriter, r *http.Request) {
	if !h.requireAnyStaff(w, r) {
		return
	}

	var req CreateRubricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	res, err := h.q.CreateRubric(r.Context(), db.CreateRubricParams{
		Title:       req.Title,
		Department:  sql.NullString{String: req.Department, Valid: req.Department != ""},
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":  "rubric created",
		"rubricId": newID,
	})
}

func (h *AppHandler) UpdateRubric(w http.ResponseWriter, r *http.Request) {
	if !h.requireAnyStaff(w, r) {
		return
	}

	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rubric id")
		return
	}

	var req CreateRubricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.q.UpdateRubric(r.Context(), db.UpdateRubricParams{
		Title:       req.Title,
		Department:  sql.NullString{String: req.Department, Valid: req.Department != ""},
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		Rubricid:    id,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "rubric updated"})
}
