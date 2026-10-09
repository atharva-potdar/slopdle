package handler

import (
	"context"
	"net/http"

	"moodleplusplus/internal/db"
	"moodleplusplus/internal/middleware"
)

const (
	roleTeacher           = "Teacher"
	roleTeachingAssistant = "Teaching Assistant"
	roleStudent           = "Student"
)

func isStaffRole(role string) bool {
	return role == roleTeacher || role == roleTeachingAssistant
}

func (h *AppHandler) callerID(r *http.Request) int32 {
	id, _ := middleware.GetUserID(r.Context())
	return id
}

// roleInCourse returns the caller's role name in a course, or "" if not enrolled.
func (h *AppHandler) roleInCourse(ctx context.Context, userID, courseID int32) string {
	if userID == 0 || courseID == 0 {
		return ""
	}
	enr, err := h.q.GetUserEnrollmentInCourse(ctx, db.GetUserEnrollmentInCourseParams{
		Userid:   userID,
		Courseid: courseID,
	})
	if err != nil {
		return ""
	}
	return enr.Rolename
}

// requireCourseStaff allows Teacher/TA in the given course, else writes 403.
func (h *AppHandler) requireCourseStaff(w http.ResponseWriter, r *http.Request, courseID int32) bool {
	if isStaffRole(h.roleInCourse(r.Context(), h.callerID(r), courseID)) {
		return true
	}
	writeError(w, http.StatusForbidden, "forbidden: staff role required for this course")
	return false
}

// requireCourseMember allows any enrolled role in the course, else 403.
func (h *AppHandler) requireCourseMember(w http.ResponseWriter, r *http.Request, courseID int32) bool {
	switch h.roleInCourse(r.Context(), h.callerID(r), courseID) {
	case roleTeacher, roleTeachingAssistant, roleStudent:
		return true
	}
	writeError(w, http.StatusForbidden, "forbidden: not enrolled in this course")
	return false
}

// requireCourseStudent allows only a Student enrolled in the course, else 403.
func (h *AppHandler) requireCourseStudent(w http.ResponseWriter, r *http.Request, courseID int32) bool {
	if h.roleInCourse(r.Context(), h.callerID(r), courseID) == roleStudent {
		return true
	}
	writeError(w, http.StatusForbidden, "forbidden: student role required for this course")
	return false
}

// requireSelfOrCourseStaff allows the target user themself, or staff in the course.
func (h *AppHandler) requireSelfOrCourseStaff(w http.ResponseWriter, r *http.Request, targetUserID, courseID int32) bool {
	if h.callerID(r) == targetUserID {
		return true
	}
	return h.requireCourseStaff(w, r, courseID)
}

// requireAnyStaff allows users with a Teacher/TA role in at least one course.
func (h *AppHandler) requireAnyStaff(w http.ResponseWriter, r *http.Request) bool {
	enrollments, err := h.q.GetEnrollmentsByUser(r.Context(), h.callerID(r))
	if err == nil {
		for _, e := range enrollments {
			if isStaffRole(e.Rolename) {
				return true
			}
		}
	}
	writeError(w, http.StatusForbidden, "forbidden: staff role required")
	return false
}
