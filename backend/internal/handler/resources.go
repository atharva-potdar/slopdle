package handler

import (
	"encoding/json"
	"net/http"

	"moodleplusplus/internal/db"
)

type CreateCategoryRequest struct {
	CategoryName string `json:"categoryName"`
}

type CreateResourceRequest struct {
	Title   string `json:"title"`
	FileURL string `json:"fileUrl"`
}

func (h *AppHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseMember(w, r, courseID) {
		return
	}

	categories, err := h.q.ListCategoriesByCourse(r.Context(), courseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

func (h *AppHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseStaff(w, r, courseID) {
		return
	}

	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CategoryName == "" {
		writeError(w, http.StatusBadRequest, "categoryName is required")
		return
	}

	res, err := h.q.CreateCategory(r.Context(), db.CreateCategoryParams{
		Categoryname: req.CategoryName,
		Courseid:     courseID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":    "category created",
		"categoryId": newID,
	})
}

func (h *AppHandler) ListResourcesByCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}

	category, err := h.q.GetCategoryByID(r.Context(), categoryID)
	if err != nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}
	if !h.requireCourseMember(w, r, category.Courseid) {
		return
	}

	resources, err := h.q.ListResourcesByCategory(r.Context(), categoryID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resources)
}

func (h *AppHandler) ListResourcesByCourse(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if !h.requireCourseMember(w, r, courseID) {
		return
	}

	resources, err := h.q.ListResourcesByCourse(r.Context(), courseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resources)
}

func (h *AppHandler) CreateResource(w http.ResponseWriter, r *http.Request) {
	categoryID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}

	category, err := h.q.GetCategoryByID(r.Context(), categoryID)
	if err != nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}
	if !h.requireCourseStaff(w, r, category.Courseid) {
		return
	}

	var req CreateResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" || req.FileURL == "" {
		writeError(w, http.StatusBadRequest, "title and fileUrl are required")
		return
	}

	res, err := h.q.CreateResource(r.Context(), db.CreateResourceParams{
		Title:      req.Title,
		Fileurl:    req.FileURL,
		Categoryid: categoryID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":    "resource created",
		"resourceId": newID,
	})
}

func (h *AppHandler) DeleteResource(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid resource id")
		return
	}

	resource, err := h.q.GetResourceByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "resource not found")
		return
	}
	if !h.requireCourseStaff(w, r, resource.Courseid) {
		return
	}

	err = h.q.DeleteResource(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "resource deleted"})
}
