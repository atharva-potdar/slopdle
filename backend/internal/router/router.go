package router

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"moodleplusplus/internal/db"
	"moodleplusplus/internal/handler"
	"moodleplusplus/internal/middleware"
)

func NewRouter(q *db.Queries, sqlDB *sql.DB) http.Handler {
	r := chi.NewRouter()

	// Base middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(middleware.Logger)
	r.Use(middleware.Cors())
	r.Use(middleware.AuthContext)

	h := handler.NewAppHandler(q, sqlDB)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","system":"moodleplusplus"}`))
	})

	r.Route("/api", func(api chi.Router) {
		// Auth (public)
		api.Post("/login", h.Login)
		api.Post("/logout", h.Logout)
		// Session probe: returns 200 with a null user when logged out (no 401 noise)
		api.Get("/me", h.Me)

		// Everything below requires a valid session
		api.Group(func(api chi.Router) {
			api.Use(middleware.RequireAuth)

			// Users
			api.Get("/users", h.ListUsers)
			api.Get("/users/{id}", h.GetUser)
			api.Put("/users/{id}", h.UpdateUserProfile)
			api.Post("/users/{id}/photos", h.AddUserPhoto)
			api.Delete("/users/{id}/photos", h.DeleteUserPhoto)

			// Courses
			api.Get("/courses", h.ListCourses)
			api.Post("/courses", h.CreateCourse)
			api.Get("/courses/{id}", h.GetCourse)
			api.Get("/courses/{id}/prerequisites", h.GetCoursePrerequisites)
			api.Post("/courses/{id}/prerequisites", h.AddCoursePrerequisite)
			api.Delete("/courses/{id}/prerequisites/{reqId}", h.DeleteCoursePrerequisite)

			// Enrollments & Batches
			api.Get("/courses/{id}/enrollments", h.GetCourseEnrollments)
			api.Post("/courses/{id}/enroll", h.EnrollUser)
			api.Delete("/courses/{id}/enroll/{userId}", h.UnenrollUser)
			api.Put("/enrollments/{enrollmentId}/batch", h.UpdateEnrollmentBatch)
			api.Get("/users/{userId}/enrollments", h.GetUserEnrollments)

			api.Get("/courses/{id}/batches", h.ListBatches)
			api.Post("/courses/{id}/batches", h.CreateBatch)
			api.Delete("/batches/{id}", h.DeleteBatch)

			// Resource Center
			api.Get("/courses/{id}/categories", h.ListCategories)
			api.Post("/courses/{id}/categories", h.CreateCategory)
			api.Get("/courses/{id}/resources", h.ListResourcesByCourse)
			api.Get("/categories/{id}/resources", h.ListResourcesByCategory)
			api.Post("/categories/{id}/resources", h.CreateResource)
			api.Delete("/resources/{id}", h.DeleteResource)

			// Assignments
			api.Get("/courses/{id}/assignments", h.ListCourseAssignments)
			api.Post("/courses/{id}/assignments", h.CreateAssignment)
			api.Get("/assignments/{id}", h.GetAssignment)
			api.Put("/assignments/{id}", h.UpdateAssignment)
			api.Delete("/assignments/{id}", h.DeleteAssignment)
			api.Get("/assignments/{id}/pending", h.GetPendingStudentsForAssignment)
			api.Get("/courses/{id}/pending-students", h.GetPendingStudentsForCourse)
			api.Get("/students/{userId}/pending-assignments", h.GetPendingAssignmentsForStudent)

			// Submissions & Evaluations
			api.Post("/assignments/{id}/submit", h.SubmitAssignment)
			api.Get("/assignments/{id}/submissions", h.ListSubmissions)
			api.Get("/submissions/{id}", h.GetSubmission)
			api.Post("/submissions/{id}/evaluate", h.EvaluateSubmission)
			api.Get("/courses/{id}/submissions/report", h.GetSubmissionReport)       // Case Study Query 3
			api.Get("/courses/{id}/students/{studentId}/grades", h.GetStudentGrades) // Case Study Query 1

			// Q&A System
			api.Get("/courses/{id}/questions", h.ListQuestions)
			api.Post("/courses/{id}/questions", h.CreateQuestion)
			api.Get("/questions/{id}", h.GetQuestion)
			api.Post("/questions/{id}/answers", h.CreateAnswer)
			api.Put("/answers/{id}/upvote", h.UpvoteAnswer)
			api.Put("/answers/{id}/official", h.MarkOfficialAnswer)

			// Attendance & Class Sessions
			api.Get("/batches/{id}/sessions", h.ListBatchSessions)
			api.Post("/batches/{id}/sessions", h.CreateClassSession)
			api.Get("/courses/{id}/sessions", h.ListCourseSessions)
			api.Post("/sessions/{id}/attendance", h.RecordAttendance)
			api.Post("/sessions/{id}/attendance/bulk", h.RecordBulkAttendance)
			api.Get("/sessions/{id}/attendance", h.GetSessionAttendance)
			api.Get("/courses/{id}/students/{studentId}/attendance", h.GetStudentCourseAttendance)

			// Notifications
			api.Get("/notifications", h.GetNotifications)
			api.Post("/notifications", h.CreateNotification)
			api.Put("/notifications/{id}/read", h.MarkNotificationRead)
			api.Put("/notifications/read-all", h.MarkAllNotificationsRead)

			// Rubrics
			api.Get("/rubrics", h.ListRubrics)
			api.Post("/rubrics", h.CreateRubric)
			api.Get("/rubrics/{id}", h.GetRubric)
			api.Put("/rubrics/{id}", h.UpdateRubric)
		})
	})

	return r
}
