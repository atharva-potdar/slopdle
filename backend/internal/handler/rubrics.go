package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	"moodleplusplus/internal/db"
)

type CreateRubricRequest struct {
	Title       string `json:"title"`
	Department  string `json:"department"`
	Description string `json:"description"`
}

type CriterionInput struct {
	CriterionID *int32  `json:"criterionId"`
	Title       string  `json:"title"`
	MaxPoints   float64 `json:"maxPoints"`
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

func (h *AppHandler) ListRubricCriteria(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rubric id")
		return
	}

	criteria, err := h.q.ListRubricCriteria(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, criteria)
}

type SetCriteriaRequest struct {
	Criteria []CriterionInput `json:"criteria"`
}

// SetRubricCriteria replaces a rubric's criterion set in one call:
// existing IDs are updated, new entries inserted, absent IDs deleted
// (their scores cascade). Staff only.
func (h *AppHandler) SetRubricCriteria(w http.ResponseWriter, r *http.Request) {
	if !h.requireAnyStaff(w, r) {
		return
	}

	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rubric id")
		return
	}
	if _, err := h.q.GetRubricByID(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "rubric not found")
		return
	}

	var req SetCriteriaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	for _, c := range req.Criteria {
		if c.Title == "" || c.MaxPoints <= 0 {
			writeError(w, http.StatusBadRequest, "each criterion needs a title and maxPoints > 0")
			return
		}
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()
	qtx := h.q.WithTx(tx)

	existing, err := qtx.ListRubricCriteria(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	keep := make(map[int32]bool, len(req.Criteria))
	for _, c := range req.Criteria {
		if c.CriterionID != nil {
			keep[*c.CriterionID] = true
		}
	}
	for _, e := range existing {
		if !keep[e.Criterionid] {
			if err := qtx.DeleteCriterion(r.Context(), e.Criterionid); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}
	for _, c := range req.Criteria {
		if c.CriterionID != nil {
			if err := qtx.UpdateCriterion(r.Context(), db.UpdateCriterionParams{
				Title:       c.Title,
				Maxpoints:   fmt.Sprintf("%.2f", c.MaxPoints),
				Criterionid: *c.CriterionID,
			}); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		} else {
			if _, err := qtx.CreateCriterion(r.Context(), db.CreateCriterionParams{
				Rubricid:  id,
				Title:     c.Title,
				Maxpoints: fmt.Sprintf("%.2f", c.MaxPoints),
			}); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "criteria saved"})
}

// normalizeCriterionTotal scales a criterion sum to /100 by the rubric's
// MaxPoints total, so rubrics totaling other than 100 still grade /100.
func normalizeCriterionTotal(sum, maxTotal float64) float64 {
	if maxTotal <= 0 {
		return 0
	}
	return math.Round(sum/maxTotal*100*100) / 100
}
