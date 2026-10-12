package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"moodleplusplus/internal/db"
	"moodleplusplus/internal/middleware"
)

type CreateNotificationRequest struct {
	UserID  int32  `json:"userId"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (h *AppHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	ctxUserID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// A user may only read their own notifications; the query param is
	// accepted for convenience but must match the session user.
	if qUser := r.URL.Query().Get("userId"); qUser != "" {
		if id, err := strconv.ParseInt(qUser, 10, 32); err != nil || int32(id) != ctxUserID {
			writeError(w, http.StatusForbidden, "forbidden: can only view own notifications")
			return
		}
	}

	notifs, err := h.q.GetUserNotifications(r.Context(), ctxUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, notifs)
}

func (h *AppHandler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	if !h.requireAnyStaff(w, r) {
		return
	}

	var req CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == 0 || req.Message == "" {
		writeError(w, http.StatusBadRequest, "userId and message are required")
		return
	}

	res, err := h.q.CreateNotification(r.Context(), db.CreateNotificationParams{
		Message: req.Message,
		Type:    sql.NullString{String: req.Type, Valid: req.Type != ""},
		Userid:  req.UserID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":        "notification created",
		"notificationId": newID,
	})
}

func (h *AppHandler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	notifID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification id")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if qUser := r.URL.Query().Get("userId"); qUser != "" {
		if id, err := strconv.ParseInt(qUser, 10, 32); err != nil || int32(id) != userID {
			writeError(w, http.StatusForbidden, "forbidden: can only modify own notifications")
			return
		}
	}

	err = h.q.MarkNotificationAsRead(r.Context(), db.MarkNotificationAsReadParams{
		Notificationid: notifID,
		Userid:         userID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "notification marked as read"})
}

func (h *AppHandler) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if qUser := r.URL.Query().Get("userId"); qUser != "" {
		if id, err := strconv.ParseInt(qUser, 10, 32); err != nil || int32(id) != userID {
			writeError(w, http.StatusForbidden, "forbidden: can only modify own notifications")
			return
		}
	}

	err := h.q.MarkAllNotificationsAsRead(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "all notifications marked as read"})
}
