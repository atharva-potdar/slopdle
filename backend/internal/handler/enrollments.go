package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"moodleplusplus/internal/db"
)

type EnrollRequest struct {
	UserID  int32 `json:"userId"`
	RoleID  int32 `json:"roleId"`
	BatchID *int32 `json:"batchId"`
}

type UpdateBatchRequest struct {
	BatchID *int32 `json:"batchId"`
}

func (h *AppHandler) GetCourseEnrollments(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseStaff(w, r, courseID) {
		return
	}

	enrollments, err := h.q.GetEnrollmentsByCourse(r.Context(), courseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, enrollments)
}

func (h *AppHandler) GetUserEnrollments(w http.ResponseWriter, r *http.Request) {
	userID, err := parseID(r, "userId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if h.callerID(r) != userID && !h.requireAnyStaff(w, r) {
		return
	}

	enrollments, err := h.q.GetEnrollmentsByUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, enrollments)
}

func (h *AppHandler) EnrollUser(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseStaff(w, r, courseID) {
		return
	}

	var req EnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == 0 || req.RoleID == 0 {
		writeError(w, http.StatusBadRequest, "userId and roleId are required")
		return
	}

	// If enrolling as a student (RoleID == 2), enforce course prerequisites!
	if req.RoleID == 2 {
		prereqs, err := h.q.GetCoursePrerequisites(r.Context(), courseID)
		if err == nil && len(prereqs) > 0 {
			for _, p := range prereqs {
				// Check student's grade in required course
				avgScore, err := h.q.CheckPrerequisiteGrade(r.Context(), db.CheckPrerequisiteGradeParams{
					Userid:   req.UserID,
					Courseid: p.Requiredcourseid,
				})
				scoreVal, ok := parseFloat(avgScore)
				if err != nil || !ok {
					writeError(w, http.StatusForbidden, fmt.Sprintf("Prerequisite not met: Must complete %s (%s) first", p.Requiredcoursecode, p.Requiredcoursetitle))
					return
				}

				if p.Minpassinggrade.Valid {
					minGrade, _ := strconv.ParseFloat(p.Minpassinggrade.String, 64)
					if scoreVal < minGrade {
						writeError(w, http.StatusForbidden, fmt.Sprintf("Prerequisite grade requirement not met: %s requires %.2f minimum grade, but current average is %.2f", p.Requiredcoursecode, minGrade, scoreVal))
						return
					}
				}
			}
		}
	}

	var batchID sql.NullInt32
	if req.BatchID != nil {
		batchID = sql.NullInt32{Int32: *req.BatchID, Valid: true}
	}

	res, err := h.q.EnrollUser(r.Context(), db.EnrollUserParams{
		Userid:   req.UserID,
		Courseid: courseID,
		Roleid:   req.RoleID,
		Batchid:  batchID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to enroll: %v", err))
		return
	}

	enrollmentID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":      "enrolled successfully",
		"enrollmentId": enrollmentID,
	})
}

func (h *AppHandler) UnenrollUser(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseStaff(w, r, courseID) {
		return
	}
	userID, err := parseID(r, "userId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	err = h.q.UnenrollUser(r.Context(), db.UnenrollUserParams{
		Userid:   userID,
		Courseid: courseID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "unenrolled successfully"})
}

func (h *AppHandler) UpdateEnrollmentBatch(w http.ResponseWriter, r *http.Request) {
	enrollmentID, err := parseID(r, "enrollmentId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid enrollment id")
		return
	}

	enrollment, err := h.q.GetEnrollmentByID(r.Context(), enrollmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "enrollment not found")
		return
	}
	if !h.requireCourseStaff(w, r, enrollment.Courseid) {
		return
	}

	var req UpdateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var batchID sql.NullInt32
	if req.BatchID != nil {
		batchID = sql.NullInt32{Int32: *req.BatchID, Valid: true}
	}

	err = h.q.UpdateEnrollmentBatch(r.Context(), db.UpdateEnrollmentBatchParams{
		Batchid:      batchID,
		Enrollmentid: enrollmentID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "batch updated"})
}
