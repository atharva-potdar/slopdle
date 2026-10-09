package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"moodleplusplus/internal/db"
)

type SubmitRequest struct {
	UserID      *int32 `json:"userId"`
	ContentData string `json:"contentData"`
}

type EvaluateRequest struct {
	Score    float64 `json:"score"`
	Feedback string  `json:"feedback"`
	GraderID *int32  `json:"graderId"`
}

func (h *AppHandler) SubmitAssignment(w http.ResponseWriter, r *http.Request) {
	assignmentID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid assignment id")
		return
	}

	assignment, err := h.q.GetAssignmentByID(r.Context(), assignmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "assignment not found")
		return
	}
	if !h.requireCourseStudent(w, r, assignment.Courseid) {
		return
	}

	var req SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ContentData) == "" {
		writeError(w, http.StatusBadRequest, "contentData is required")
		return
	}

	userID := h.callerID(r)

	// Verify format requirement if specified on assignment
	if assignment.Formatrequired.Valid {
		fmtReq := strings.ToUpper(assignment.Formatrequired.String)
		if strings.Contains(fmtReq, "GITHUB") || strings.Contains(fmtReq, "REPO") {
			if !strings.HasPrefix(strings.ToLower(req.ContentData), "http") || !strings.Contains(strings.ToLower(req.ContentData), "github.com") {
				writeError(w, http.StatusBadRequest, "Submission requires a valid GitHub repository URL (Format: GitHub)")
				return
			}
		}
	}

	res, err := h.q.CreateSubmission(r.Context(), db.CreateSubmissionParams{
		Contentdata:  req.ContentData,
		Assignmentid: assignmentID,
		Userid:       userID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	subID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":      "submitted successfully",
		"submissionId": subID,
	})
}

func (h *AppHandler) ListSubmissions(w http.ResponseWriter, r *http.Request) {
	assignmentID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid assignment id")
		return
	}

	assignment, err := h.q.GetAssignmentByID(r.Context(), assignmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "assignment not found")
		return
	}
	if !h.requireCourseStaff(w, r, assignment.Courseid) {
		return
	}

	submissions, err := h.q.ListSubmissionsByAssignment(r.Context(), assignmentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, submissions)
}

func (h *AppHandler) GetSubmission(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid submission id")
		return
	}

	sub, err := h.q.GetSubmissionByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "submission not found")
		return
	}
	if !h.requireSelfOrCourseStaff(w, r, sub.Userid, sub.Courseid) {
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (h *AppHandler) EvaluateSubmission(w http.ResponseWriter, r *http.Request) {
	submissionID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid submission id")
		return
	}

	sub, err := h.q.GetSubmissionByID(r.Context(), submissionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "submission not found")
		return
	}
	if !h.requireCourseStaff(w, r, sub.Courseid) {
		return
	}

	var req EvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Score < 0 {
		writeError(w, http.StatusBadRequest, "valid score (>= 0) required")
		return
	}

	graderID := h.callerID(r)

	err = h.q.CreateOrUpdateEvaluation(r.Context(), db.CreateOrUpdateEvaluationParams{
		Submissionid: submissionID,
		Score:        fmt.Sprintf("%.2f", req.Score),
		Feedback:     sql.NullString{String: req.Feedback, Valid: req.Feedback != ""},
		Userid:       graderID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Notify student of graded submission
	msg := fmt.Sprintf("Your submission for '%s' has been evaluated: Score %.2f", sub.Assignmenttitle, req.Score)
	h.q.CreateNotification(r.Context(), db.CreateNotificationParams{
		Message: msg,
		Type:    sql.NullString{String: "Grade", Valid: true},
		Userid:  sub.Userid,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "evaluation saved"})
}

// Case Study Query 3: Submission report with timeliness and grader name
func (h *AppHandler) GetSubmissionReport(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseStaff(w, r, courseID) {
		return
	}

	report, err := h.q.GetSubmissionReport(r.Context(), courseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type ReportItem struct {
		AssignmentID   int32   `json:"assignmentId"`
		AssignmentName string  `json:"assignmentName"`
		StudentName    string  `json:"studentName"`
		SubmissionID   int32   `json:"submissionId"`
		SubmissionTime *string `json:"submissionTime"`
		Deadline       string  `json:"deadline"`
		Timeliness     string  `json:"timeliness"`
		Score          *string `json:"score"`
		GradedBy       string  `json:"gradedBy"`
	}

	items := make([]ReportItem, 0, len(report))
	for _, row := range report {
		var subTime *string
		if row.Submissiontime.Valid {
			t := row.Submissiontime.Time.Format("2006-01-02 15:04:05")
			subTime = &t
		}
		var score *string
		if row.Score.Valid {
			s := row.Score.String
			score = &s
		}
		graderStr := "Ungraded"
		switch g := row.Gradedby.(type) {
		case []byte:
			graderStr = string(g)
		case string:
			graderStr = g
		}

		items = append(items, ReportItem{
			AssignmentID:   row.Assignmentid,
			AssignmentName: row.Assignmentname,
			StudentName:    row.Studentname,
			SubmissionID:   row.Submissionid,
			SubmissionTime: subTime,
			Deadline:       row.Deadline.Format("2006-01-02 15:04:05"),
			Timeliness:     row.Timeliness,
			Score:          score,
			GradedBy:       graderStr,
		})
	}

	writeJSON(w, http.StatusOK, items)
}

// Case Study Query 1: Student assignment scores and feedback
func (h *AppHandler) GetStudentGrades(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	studentID, err := parseID(r, "studentId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid student id")
		return
	}
	if !h.requireSelfOrCourseStaff(w, r, studentID, courseID) {
		return
	}

	grades, err := h.q.GetStudentGradesInCourse(r.Context(), db.GetStudentGradesInCourseParams{
		Userid:   studentID,
		Courseid: courseID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, grades)
}
