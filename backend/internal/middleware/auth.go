package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

type contextKey string

const (
	UserIDKey contextKey = "userID"
)

type Session struct {
	UserID    int32
	ExpiresAt time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

var Store = &SessionStore{
	sessions: make(map[string]Session),
}

func (s *SessionStore) Create(userID int32) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)
	s.sessions[token] = Session{
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	return token
}

func (s *SessionStore) Get(token string) (int32, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, exists := s.sessions[token]
	if !exists || time.Now().After(sess.ExpiresAt) {
		return 0, false
	}
	return sess.UserID, true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func AuthContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err == nil && cookie.Value != "" {
			if userID, ok := Store.Get(cookie.Value); ok {
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(UserIDKey).(int32)
		if !ok || userID == 0 {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserID(ctx context.Context) (int32, bool) {
	userID, ok := ctx.Value(UserIDKey).(int32)
	return userID, ok && userID > 0
}
