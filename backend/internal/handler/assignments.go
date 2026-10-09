package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"moodleplusplus/internal/db"
)

type CreateAssignmentRequest struct {
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Deadline       time.Time `json:"deadline"`
	FormatRequired string    `json:"formatRequired"`
	RubricID       *int32    `json:"rubricId"`
}

func (h *AppHandler) ListCourseAssignments(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseMember(w, r, courseID) {
		return
	}

	assignments, err := h.q.ListAssignmentsByCourse(r.Context(), courseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, assignments)
}

func (h *AppHandler) GetAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid assignment id")
		return
	}

	assignment, err := h.q.GetAssignmentByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "assignment not found")
		return
	}
	if !h.requireCourseMember(w, r, assignment.Courseid) {
		return
	}
	writeJSON(w, http.StatusOK, assignment)
}

func (h *AppHandler) CreateAssignment(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseStaff(w, r, courseID) {
		return
	}

	var req CreateAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" || req.Deadline.IsZero() {
		writeError(w, http.StatusBadRequest, "title and deadline are required")
		return
	}

	var rubricID sql.NullInt32
	if req.RubricID != nil {
		rubricID = sql.NullInt32{Int32: *req.RubricID, Valid: true}
	}

	res, err := h.q.CreateAssignment(r.Context(), db.CreateAssignmentParams{
		Title:          req.Title,
		Description:    sql.NullString{String: req.Description, Valid: req.Description != ""},
		Deadline:       req.Deadline,
		Formatrequired: sql.NullString{String: req.FormatRequired, Valid: req.FormatRequired != ""},
		Courseid:       courseID,
		Rubricid:       rubricID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":      "assignment created",
		"assignmentId": newID,
	})
}

func (h *AppHandler) UpdateAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid assignment id")
		return
	}

	assignment, err := h.q.GetAssignmentByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "assignment not found")
		return
	}
	if !h.requireCourseStaff(w, r, assignment.Courseid) {
		return
	}

	var req CreateAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var rubricID sql.NullInt32
	if req.RubricID != nil {
		rubricID = sql.NullInt32{Int32: *req.RubricID, Valid: true}
	}

	err = h.q.UpdateAssignment(r.Context(), db.UpdateAssignmentParams{
		Title:          req.Title,
		Description:    sql.NullString{String: req.Description, Valid: req.Description != ""},
		Deadline:       req.Deadline,
		Formatrequired: sql.NullString{String: req.FormatRequired, Valid: req.FormatRequired != ""},
		Rubricid:       rubricID,
		Assignmentid:   id,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "assignment updated"})
}

func (h *AppHandler) DeleteAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid assignment id")
		return
	}

	assignment, err := h.q.GetAssignmentByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "assignment not found")
		return
	}
	if !h.requireCourseStaff(w, r, assignment.Courseid) {
		return
	}

	err = h.q.DeleteAssignment(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "assignment deleted"})
}

// Case Study Query 2: Students enrolled in course who have not submitted
func (h *AppHandler) GetPendingStudentsForAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid assignment id")
		return
	}

	assignment, err := h.q.GetAssignmentByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "assignment not found")
		return
	}
	if !h.requireCourseStaff(w, r, assignment.Courseid) {
		return
	}

	pending, err := h.q.GetPendingStudentsForAssignment(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pending)
}

func (h *AppHandler) GetPendingAssignmentsForStudent(w http.ResponseWriter, r *http.Request) {
	userID, err := parseID(r, "userId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if h.callerID(r) != userID && !h.requireAnyStaff(w, r) {
		return
	}

	pending, err := h.q.GetPendingAssignmentsForStudent(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pending)
}

func (h *AppHandler) GetPendingStudentsForCourse(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseStaff(w, r, id) {
		return
	}

	pending, err := h.q.GetPendingStudentsForCourse(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pending)
}
