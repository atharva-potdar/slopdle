package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"moodleplusplus/internal/db"
	"moodleplusplus/internal/middleware"
)

type CreateQuestionRequest struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID *int32 `json:"userId"`
}

type CreateAnswerRequest struct {
	Body   string `json:"body"`
	UserID *int32 `json:"userId"`
}

type OfficialAnswerRequest struct {
	IsOfficial bool `json:"isOfficial"`
}

func (h *AppHandler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseMember(w, r, courseID) {
		return
	}

	questions, err := h.q.ListQuestionsByCourse(r.Context(), courseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, questions)
}

func (h *AppHandler) GetQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid question id")
		return
	}

	question, err := h.q.GetQuestionByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "question not found")
		return
	}
	if !h.requireCourseMember(w, r, question.Courseid) {
		return
	}

	answers, _ := h.q.ListAnswersByQuestion(r.Context(), db.ListAnswersByQuestionParams{
		Userid:     h.callerID(r),
		Questionid: id,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"question": question,
		"answers":  answers,
	})
}

func (h *AppHandler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseMember(w, r, courseID) {
		return
	}

	var req CreateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" || req.Body == "" {
		writeError(w, http.StatusBadRequest, "title and body are required")
		return
	}

	userID := int32(0)
	if ctxUserID, ok := middleware.GetUserID(r.Context()); ok {
		userID = ctxUserID
	} else if req.UserID != nil {
		userID = *req.UserID
	}

	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "user authentication or userId required")
		return
	}

	res, err := h.q.CreateQuestion(r.Context(), db.CreateQuestionParams{
		Title:    req.Title,
		Body:     req.Body,
		Courseid: courseID,
		Userid:   userID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":    "question created",
		"questionId": newID,
	})
}

func (h *AppHandler) CreateAnswer(w http.ResponseWriter, r *http.Request) {
	questionID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid question id")
		return
	}

	question, err := h.q.GetQuestionByID(r.Context(), questionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "question not found")
		return
	}
	if !h.requireCourseMember(w, r, question.Courseid) {
		return
	}

	var req CreateAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	userID := int32(0)
	if ctxUserID, ok := middleware.GetUserID(r.Context()); ok {
		userID = ctxUserID
	} else if req.UserID != nil {
		userID = *req.UserID
	}

	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "user authentication or userId required")
		return
	}

	res, err := h.q.CreateAnswer(r.Context(), db.CreateAnswerParams{
		Body:       req.Body,
		Questionid: questionID,
		Userid:     userID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newID, _ := res.LastInsertId()

	// Notify question author
	if question.Userid != userID {
		h.q.CreateNotification(r.Context(), db.CreateNotificationParams{
			Message: fmt.Sprintf("Your question '%s' received a new answer.", question.Title),
			Type:    sql.NullString{String: "Q&A Reply", Valid: true},
			Userid:  question.Userid,
		})
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message":  "answer posted",
		"answerId": newID,
	})
}

func (h *AppHandler) UpvoteAnswer(w http.ResponseWriter, r *http.Request) {
	answerID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid answer id")
		return
	}

	answer, err := h.q.GetAnswerByID(r.Context(), answerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "answer not found")
		return
	}
	if !h.requireCourseMember(w, r, answer.Courseid) {
		return
	}

	userID := h.callerID(r)

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()
	qtx := h.q.WithTx(tx)

	count, err := qtx.HasAnswerUpvote(r.Context(), db.HasAnswerUpvoteParams{
		Answerid: answerID,
		Userid:   userID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	upvoted := count == 0
	if upvoted {
		err = qtx.AddAnswerUpvote(r.Context(), db.AddAnswerUpvoteParams{
			Answerid: answerID,
			Userid:   userID,
		})
		if err == nil {
			err = qtx.IncrementAnswerUpvotes(r.Context(), answerID)
		}
	} else {
		err = qtx.RemoveAnswerUpvote(r.Context(), db.RemoveAnswerUpvoteParams{
			Answerid: answerID,
			Userid:   userID,
		})
		if err == nil {
			err = qtx.DecrementAnswerUpvotes(r.Context(), answerID)
		}
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":   "upvote toggled",
		"upvoted":   upvoted,
		"upvotes":   answer.Upvotes.Int32 + boolToInt(upvoted) - boolToInt(!upvoted),
	})
}

func boolToInt(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

func (h *AppHandler) MarkOfficialAnswer(w http.ResponseWriter, r *http.Request) {
	answerID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid answer id")
		return
	}

	answer, err := h.q.GetAnswerByID(r.Context(), answerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "answer not found")
		return
	}
	if !h.requireCourseStaff(w, r, answer.Courseid) {
		return
	}

	var req OfficialAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.q.MarkOfficialAnswer(r.Context(), db.MarkOfficialAnswerParams{
		Isofficialanswer: sql.NullBool{Bool: req.IsOfficial, Valid: true},
		Answerid:         answerID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "answer status updated"})
}
