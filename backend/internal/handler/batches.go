package handler

import (
	"encoding/json"
	"net/http"

	"moodleplusplus/internal/db"
)

type CreateBatchRequest struct {
	BatchName string `json:"batchName"`
}

func (h *AppHandler) ListBatches(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseMember(w, r, courseID) {
		return
	}

	batches, err := h.q.ListBatchesByCourse(r.Context(), courseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, batches)
}

func (h *AppHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseStaff(w, r, courseID) {
		return
	}

	var req CreateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BatchName == "" {
		writeError(w, http.StatusBadRequest, "batchName is required")
		return
	}

	res, err := h.q.CreateBatch(r.Context(), db.CreateBatchParams{
		Batchname: req.BatchName,
		Courseid:  courseID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "batch created",
		"batchId": newID,
	})
}

func (h *AppHandler) DeleteBatch(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid batch id")
		return
	}

	batch, err := h.q.GetBatchByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "batch not found")
		return
	}
	if !h.requireCourseStaff(w, r, batch.Courseid) {
		return
	}

	err = h.q.DeleteBatch(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "batch deleted"})
}
