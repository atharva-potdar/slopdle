package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"moodleplusplus/internal/db"
)

type CreateSessionRequest struct {
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	RoomLocation string    `json:"roomLocation"`
}

type RecordAttendanceRequest struct {
	UserID int32  `json:"userId"`
	Status string `json:"status"` // 'Present', 'Absent', 'Excused'
}

type BulkAttendanceRequest struct {
	Records []RecordAttendanceRequest `json:"records"`
}

func (h *AppHandler) ListBatchSessions(w http.ResponseWriter, r *http.Request) {
	batchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid batch id")
		return
	}

	batch, err := h.q.GetBatchByID(r.Context(), batchID)
	if err != nil {
		writeError(w, http.StatusNotFound, "batch not found")
		return
	}
	if !h.requireCourseMember(w, r, batch.Courseid) {
		return
	}

	sessions, err := h.q.ListSessionsByBatch(r.Context(), batchID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (h *AppHandler) ListCourseSessions(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseMember(w, r, courseID) {
		return
	}

	sessions, err := h.q.ListSessionsByCourse(r.Context(), courseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (h *AppHandler) CreateClassSession(w http.ResponseWriter, r *http.Request) {
	batchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid batch id")
		return
	}

	batch, err := h.q.GetBatchByID(r.Context(), batchID)
	if err != nil {
		writeError(w, http.StatusNotFound, "batch not found")
		return
	}
	if !h.requireCourseStaff(w, r, batch.Courseid) {
		return
	}

	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.StartTime.IsZero() || req.EndTime.IsZero() {
		writeError(w, http.StatusBadRequest, "startTime and endTime are required")
		return
	}

	if !req.EndTime.After(req.StartTime) {
		writeError(w, http.StatusBadRequest, "endTime must be after startTime")
		return
	}

	res, err := h.q.CreateClassSession(r.Context(), db.CreateClassSessionParams{
		Starttime:    req.StartTime,
		Endtime:      req.EndTime,
		Roomlocation: sql.NullString{String: req.RoomLocation, Valid: req.RoomLocation != ""},
		Batchid:      batchID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":   "session created",
		"sessionId": newID,
	})
}

func (h *AppHandler) RecordAttendance(w http.ResponseWriter, r *http.Request) {
	sessionID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return
	}

	session, err := h.q.GetSessionByID(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if !h.requireCourseStaff(w, r, session.Courseid) {
		return
	}

	var req RecordAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == 0 || req.Status == "" {
		writeError(w, http.StatusBadRequest, "userId and status ('Present','Absent','Excused') are required")
		return
	}

	err = h.q.RecordAttendance(r.Context(), db.RecordAttendanceParams{
		Sessionid: sessionID,
		Userid:    req.UserID,
		Status:    req.Status,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "attendance recorded"})
}

func (h *AppHandler) RecordBulkAttendance(w http.ResponseWriter, r *http.Request) {
	sessionID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return
	}

	session, err := h.q.GetSessionByID(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if !h.requireCourseStaff(w, r, session.Courseid) {
		return
	}

	var req BulkAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Records) == 0 {
		writeError(w, http.StatusBadRequest, "records array required")
		return
	}

	for _, rec := range req.Records {
		_ = h.q.RecordAttendance(r.Context(), db.RecordAttendanceParams{
			Sessionid: sessionID,
			Userid:    rec.UserID,
			Status:    rec.Status,
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "bulk attendance recorded"})
}

func (h *AppHandler) GetSessionAttendance(w http.ResponseWriter, r *http.Request) {
	sessionID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return
	}

	session, err := h.q.GetSessionByID(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if !h.requireCourseStaff(w, r, session.Courseid) {
		return
	}

	attendanceList, err := h.q.GetSessionAttendance(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, attendanceList)
}

func (h *AppHandler) GetStudentCourseAttendance(w http.ResponseWriter, r *http.Request) {
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

	timeline, err := h.q.GetUserAttendanceInCourse(r.Context(), db.GetUserAttendanceInCourseParams{
		Courseid: courseID,
		Userid:   studentID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	summary, _ := h.q.GetUserAttendanceSummary(r.Context(), db.GetUserAttendanceSummaryParams{
		Courseid: courseID,
		Userid:   studentID,
	})

	present := parseCount(summary.Presentcount)
	absent := parseCount(summary.Absentcount)
	excused := parseCount(summary.Excusedcount)

	percentage := 0.0
	if summary.Totalsessions > 0 {
		percentage = (float64(present) / float64(summary.Totalsessions)) * 100.0
	}

	type SessionItem struct {
		SessionID        int32  `json:"sessionId"`
		StartTime        string `json:"startTime"`
		EndTime          string `json:"endTime"`
		RoomLocation     string `json:"roomLocation"`
		BatchName        string `json:"batchName"`
		AttendanceStatus string `json:"attendanceStatus"`
	}

	sessionItems := make([]SessionItem, 0, len(timeline))
	for _, s := range timeline {
		statusStr := "Unmarked"
		switch st := s.Attendancestatus.(type) {
		case []byte:
			statusStr = string(st)
		case string:
			statusStr = st
		}
		sessionItems = append(sessionItems, SessionItem{
			SessionID:        s.Sessionid,
			StartTime:        s.Starttime.Format("2006-01-02 15:04"),
			EndTime:          s.Endtime.Format("15:04"),
			RoomLocation:     s.Roomlocation.String,
			BatchName:        s.Batchname,
			AttendanceStatus: statusStr,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"percentage":    fmt.Sprintf("%.1f%%", percentage),
		"percentageRaw": percentage,
		"totalSessions": summary.Totalsessions,
		"presentCount":  present,
		"absentCount":   absent,
		"excusedCount":  excused,
		"sessions":      sessionItems,
	})
}

func parseCount(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int32:
		return int64(val)
	case []byte:
		n, _ := strconv.ParseInt(string(val), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(val, 10, 64)
		return n
	default:
		return 0
	}
}
