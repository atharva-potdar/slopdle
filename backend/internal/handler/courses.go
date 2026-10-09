package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"moodleplusplus/internal/db"
)

type CreateCourseRequest struct {
	CourseCode  string `json:"courseCode"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type AddPrereqRequest struct {
	RequiredCourseID int32   `json:"requiredCourseId"`
	MinPassingGrade  float64 `json:"minPassingGrade"`
}

func (h *AppHandler) ListCourses(w http.ResponseWriter, r *http.Request) {
	courses, err := h.q.ListCoursesByUser(r.Context(), h.callerID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, courses)
}

func (h *AppHandler) GetCourse(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseMember(w, r, id) {
		return
	}

	course, err := h.q.GetCourseByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "course not found")
		return
	}

	prereqs, _ := h.q.GetCoursePrerequisites(r.Context(), id)

	writeJSON(w, http.StatusOK, map[string]any{
		"course":        course,
		"prerequisites": prereqs,
	})
}

func (h *AppHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	if !h.requireAnyStaff(w, r) {
		return
	}

	var req CreateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CourseCode == "" || req.Title == "" {
		writeError(w, http.StatusBadRequest, "courseCode and title are required")
		return
	}

	res, err := h.q.CreateCourse(r.Context(), db.CreateCourseParams{
		Coursecode:  req.CourseCode,
		Title:       req.Title,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":  "course created",
		"courseId": newID,
	})
}

func (h *AppHandler) GetCoursePrerequisites(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseMember(w, r, id) {
		return
	}

	prereqs, err := h.q.GetCoursePrerequisites(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, prereqs)
}

func (h *AppHandler) AddCoursePrerequisite(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseStaff(w, r, courseID) {
		return
	}

	var req AddPrereqRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RequiredCourseID == 0 {
		writeError(w, http.StatusBadRequest, "valid requiredCourseId required")
		return
	}

	if courseID == req.RequiredCourseID {
		writeError(w, http.StatusBadRequest, "a course cannot be a prerequisite of itself")
		return
	}

	err = h.q.AddCoursePrerequisite(r.Context(), db.AddCoursePrerequisiteParams{
		Courseid:         courseID,
		Requiredcourseid: req.RequiredCourseID,
		Minpassinggrade:  sql.NullString{String: fmt.Sprintf("%.2f", req.MinPassingGrade), Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "prerequisite added"})
}

func (h *AppHandler) DeleteCoursePrerequisite(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseStaff(w, r, courseID) {
		return
	}
	reqID, err := parseID(r, "reqId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid reqId")
		return
	}

	err = h.q.DeleteCoursePrerequisite(r.Context(), db.DeleteCoursePrerequisiteParams{
		Courseid:         courseID,
		Requiredcourseid: reqID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "prerequisite removed"})
}
