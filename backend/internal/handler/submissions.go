package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"moodleplusplus/internal/db"
)

type SubmitRequest struct {
	UserID      *int32 `json:"userId"`
	ContentData string `json:"contentData"`
}

type EvaluateRequest struct {
	Score     float64                `json:"score"`
	Feedback  string                 `json:"feedback"`
	GraderID  *int32                 `json:"graderId"`
	Criteria  []CriterionScoreInput  `json:"criteria"`
}

type CriterionScoreInput struct {
	CriterionID int32   `json:"criterionId"`
	Score       float64 `json:"score"`
	Feedback    string  `json:"feedback"`
}

// applyLatePenalty returns days late, penalty amount, and effective score.
// EffectiveScore = max(0, raw - raw*daysLate*perDay/100). Disabled,
// on-time, or missing submission time → raw score, zero penalty.
func applyLatePenalty(raw float64, submitted, deadline time.Time, submittedValid bool, perDay float64, enabled bool) (int, float64, float64) {
	if !enabled || !submittedValid || !submitted.After(deadline) {
		return 0, 0, raw
	}
	days := int(math.Ceil(submitted.Sub(deadline).Hours() / 24))
	rate := float64(days) * perDay / 100.0
	if rate > 1 {
		rate = 1
	}
	penalty := math.Round(raw*rate*100) / 100
	effective := math.Round((raw-penalty)*100) / 100
	if effective < 0 {
		effective = 0
	}
	return days, penalty, effective
}

func strptr(s string) *string { return &s }

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

	visible, err := h.q.CountAssignmentVisibleToStudent(r.Context(), db.CountAssignmentVisibleToStudentParams{
		Assignmentid: assignmentID,
		Courseid:     assignment.Courseid,
		Userid:       h.callerID(r),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if visible == 0 {
		writeError(w, http.StatusForbidden, "assignment is not assigned to your batch")
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

	// Raw score defaults to the flat input; criterion grading overrides it
	// with the normalized criterion total.
	rawScore := req.Score
	if len(req.Criteria) > 0 {
		assignment, err := h.q.GetAssignmentByID(r.Context(), sub.Assignmentid)
		if err != nil || !assignment.Rubricid.Valid {
			writeError(w, http.StatusBadRequest, "criterion grading requires an assignment rubric")
			return
		}
		criteria, err := h.q.ListRubricCriteria(r.Context(), assignment.Rubricid.Int32)
		if err != nil || len(criteria) == 0 || len(req.Criteria) != len(criteria) {
			writeError(w, http.StatusBadRequest, "must score every criterion of the rubric")
			return
		}
		maxByID := make(map[int32]float64, len(criteria))
		maxTotal := 0.0
		for _, c := range criteria {
			max, _ := parseFloat(c.Maxpoints)
			maxByID[c.Criterionid] = max
			maxTotal += max
		}
		sum := 0.0
		for _, cs := range req.Criteria {
			max, ok := maxByID[cs.CriterionID]
			if !ok || cs.Score < 0 || cs.Score > max {
				writeError(w, http.StatusBadRequest, "criterion score out of range")
				return
			}
			sum += cs.Score
		}
		rawScore = normalizeCriterionTotal(sum, maxTotal)
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()
	qtx := h.q.WithTx(tx)

	err = qtx.CreateOrUpdateEvaluation(r.Context(), db.CreateOrUpdateEvaluationParams{
		Submissionid: submissionID,
		Score:        fmt.Sprintf("%.2f", rawScore),
		Feedback:     sql.NullString{String: req.Feedback, Valid: req.Feedback != ""},
		Userid:       graderID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, cs := range req.Criteria {
		if err := qtx.UpsertCriterionScore(r.Context(), db.UpsertCriterionScoreParams{
			Submissionid: submissionID,
			Criterionid:  cs.CriterionID,
			Score:        fmt.Sprintf("%.2f", cs.Score),
			Feedback:     sql.NullString{String: cs.Feedback, Valid: cs.Feedback != ""},
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := tx.Commit(); err != nil {
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
		AssignmentID    int32   `json:"assignmentId"`
		AssignmentName  string  `json:"assignmentName"`
		StudentName     string  `json:"studentName"`
		SubmissionID    int32   `json:"submissionId"`
		SubmissionTime  *string `json:"submissionTime"`
		Content         string  `json:"content"`
		Deadline        string  `json:"deadline"`
		Timeliness      string  `json:"timeliness"`
		Score           *string `json:"score"`
		DaysLate        int     `json:"daysLate"`
		PenaltyPerDay   string  `json:"penaltyPerDay"`
		PenaltyEnabled  bool    `json:"penaltyEnabled"`
		PenaltyApplied  *string `json:"penaltyApplied"`
		EffectiveScore  *string `json:"effectiveScore"`
		GradedBy        string  `json:"gradedBy"`
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

		perDay, _ := parseFloat(row.Latepenaltyperday)
		var daysLate int
		var penaltyApplied, effectiveScore *string
		if row.Score.Valid {
			raw, _ := parseFloat(row.Score.String)
			d, p, e := applyLatePenalty(raw, row.Submissiontime.Time, row.Deadline, row.Submissiontime.Valid, perDay, row.Latepenaltyenabled)
			daysLate = d
			penaltyApplied = strptr(fmt.Sprintf("%.2f", p))
			effectiveScore = strptr(fmt.Sprintf("%.2f", e))
		} else if row.Submissiontime.Valid {
			daysLate, _, _ = applyLatePenalty(0, row.Submissiontime.Time, row.Deadline, true, perDay, row.Latepenaltyenabled)
		}

		items = append(items, ReportItem{
			AssignmentID:   row.Assignmentid,
			AssignmentName: row.Assignmentname,
			StudentName:    row.Studentname,
			SubmissionID:   row.Submissionid,
			SubmissionTime: subTime,
			Content:        row.Contentdata,
			Deadline:       row.Deadline.Format("2006-01-02 15:04:05"),
			Timeliness:     row.Timeliness,
			Score:          score,
			DaysLate:       daysLate,
			PenaltyPerDay:  row.Latepenaltyperday,
			PenaltyEnabled: row.Latepenaltyenabled,
			PenaltyApplied: penaltyApplied,
			EffectiveScore: effectiveScore,
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

	type GradeItem struct {
		FirstName          string         `json:"firstname"`
		AssignmentID       int32          `json:"assignmentid"`
		Assignment         string         `json:"assignment"`
		Score              string         `json:"score"`
		Feedback           sql.NullString `json:"feedback"`
		DaysLate           int            `json:"dayslate"`
		PenaltyPerDay      string         `json:"penaltyperday"`
		PenaltyEnabled     bool           `json:"penaltyenabled"`
		PenaltyApplied     string         `json:"penaltyapplied"`
		EffectiveScore     string         `json:"effectivescore"`
		CriterionBreakdown sql.NullString `json:"criterionbreakdown"`
	}

	items := make([]GradeItem, 0, len(grades))
	for _, g := range grades {
		raw, _ := parseFloat(g.Score)
		perDay, _ := parseFloat(g.Latepenaltyperday)
		days, penalty, effective := applyLatePenalty(raw, g.Submissiontime.Time, g.Deadline, g.Submissiontime.Valid, perDay, g.Latepenaltyenabled)
		items = append(items, GradeItem{
			FirstName:          g.Firstname,
			AssignmentID:       g.Assignmentid,
			Assignment:         g.Assignment,
			Score:              g.Score,
			Feedback:           g.Feedback,
			DaysLate:           days,
			PenaltyPerDay:      g.Latepenaltyperday,
			PenaltyEnabled:     g.Latepenaltyenabled,
			PenaltyApplied:     fmt.Sprintf("%.2f", penalty),
			EffectiveScore:     fmt.Sprintf("%.2f", effective),
			CriterionBreakdown: g.Criterionbreakdown,
		})
	}
	writeJSON(w, http.StatusOK, items)
}

// GetSubmissionCriteria returns per-criterion scores for a submission
// so re-grading can prefill sliders and students can see breakdowns.
func (h *AppHandler) GetSubmissionCriteria(w http.ResponseWriter, r *http.Request) {
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

	scores, err := h.q.ListCriterionScoresBySubmission(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, scores)
}

// ListStudentSubmissionsByCourse returns all of a student's submissions in a
// course so the UI can indicate what has already been submitted.
func (h *AppHandler) GetStudentSubmissions(w http.ResponseWriter, r *http.Request) {
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

	subs, err := h.q.ListStudentSubmissionsByCourse(r.Context(), db.ListStudentSubmissionsByCourseParams{
		Userid:   studentID,
		Courseid: courseID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

// GetUserSubmissions returns all of a user's submissions across courses
// so the dashboard can show completed work alongside pending work.
func (h *AppHandler) GetUserSubmissions(w http.ResponseWriter, r *http.Request) {
	userID, err := parseID(r, "userId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if h.callerID(r) != userID && !h.requireAnyStaff(w, r) {
		return
	}

	subs, err := h.q.ListStudentSubmissionsByUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type UserSubmissionItem struct {
		CourseCode     string       `json:"coursecode"`
		CourseTitle    string       `json:"coursetitle"`
		AssignmentID   int32        `json:"assignmentid"`
		AssignmentName string       `json:"assignmentname"`
		Deadline       time.Time    `json:"deadline"`
		SubmissionID   int32        `json:"submissionid"`
		SubmissionTime sql.NullTime `json:"submissiontime"`
		Score          sql.NullString `json:"score"`
		EffectiveScore *string      `json:"effectivescore"`
	}

	items := make([]UserSubmissionItem, 0, len(subs))
	for _, s := range subs {
		item := UserSubmissionItem{
			CourseCode:     s.Coursecode,
			CourseTitle:    s.Coursetitle,
			AssignmentID:   s.Assignmentid,
			AssignmentName: s.Assignmentname,
			Deadline:       s.Deadline,
			SubmissionID:   s.Submissionid,
			SubmissionTime: s.Submissiontime,
			Score:          s.Score,
		}
		if s.Score.Valid {
			raw, _ := parseFloat(s.Score.String)
			perDay, _ := parseFloat(s.Latepenaltyperday)
			_, _, effective := applyLatePenalty(raw, s.Submissiontime.Time, s.Deadline, s.Submissiontime.Valid, perDay, s.Latepenaltyenabled)
			item.EffectiveScore = strptr(fmt.Sprintf("%.2f", effective))
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, items)
}
