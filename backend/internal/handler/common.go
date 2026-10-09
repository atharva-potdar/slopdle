package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"moodleplusplus/internal/db"
)

type AppHandler struct {
	q  *db.Queries
	db *sql.DB
}

func NewAppHandler(q *db.Queries, sqlDB *sql.DB) *AppHandler {
	return &AppHandler{q: q, db: sqlDB}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseID(r *http.Request, param string) (int32, error) {
	val := chi.URLParam(r, param)
	id, err := strconv.ParseInt(val, 10, 32)
	return int32(id), err
}

func parseFloat(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case []byte:
		f, err := strconv.ParseFloat(string(val), 64)
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	default:
		return 0, false
	}
}
