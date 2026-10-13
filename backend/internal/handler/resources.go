package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"moodleplusplus/internal/db"
)

type CreateCategoryRequest struct {
	CategoryName string `json:"categoryName"`
}

type CreateResourceRequest struct {
	Title    string  `json:"title"`
	FileURL  string  `json:"fileUrl"`
	BatchIDs []int32 `json:"batchIds"`
}

// setResourceBatches replaces a resource's batch scope atomically.
// An empty slice means "course-wide" (no restriction).
func (h *AppHandler) setResourceBatches(ctx context.Context, resourceID int32, batchIDs []int32) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := h.q.WithTx(tx)
	if err := qtx.DeleteResourceBatches(ctx, resourceID); err != nil {
		return err
	}
	for _, bid := range batchIDs {
		if err := qtx.AddResourceBatch(ctx, db.AddResourceBatchParams{
			Resourceid: resourceID,
			Batchid:    bid,
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
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

	if h.isStaffInCourse(r.Context(), h.callerID(r), category.Courseid) {
		resources, err := h.q.ListResourcesByCategory(r.Context(), categoryID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resources)
		return
	}

	resources, err := h.q.ListResourcesByCategoryForStudent(r.Context(), db.ListResourcesByCategoryForStudentParams{
		Categoryid: categoryID,
		Userid:     h.callerID(r),
	})
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

	if h.isStaffInCourse(r.Context(), h.callerID(r), courseID) {
		resources, err := h.q.ListResourcesByCourse(r.Context(), courseID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resources)
		return
	}

	resources, err := h.q.ListResourcesByCourseForStudent(r.Context(), db.ListResourcesByCourseForStudentParams{
		Courseid: courseID,
		Userid:   h.callerID(r),
	})
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
	if err := h.setResourceBatches(r.Context(), int32(newID), req.BatchIDs); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
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

	stored := ""
	if resource.Filename.Valid && resource.Filename.String != "" {
		stored = resource.Filename.String
	}

	err = h.q.DeleteResource(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Best-effort disk sweep for file rows (link rows have no file).
	if stored != "" {
		if err := removeStoredFile(stored); err != nil {
			fmt.Printf("sweep: cannot remove resource file %s: %v\n", stored, err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "resource deleted"})
}

// UploadResource stores one PDF/ZIP file as a course resource (staff only).
// One resource = one document. Link-based resources still use CreateResource.
func (h *AppHandler) UploadResource(w http.ResponseWriter, r *http.Request) {
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

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	const resMaxMB = 25
	capBytes := int64(resMaxMB)<<20 + 1<<20
	r.Body = http.MaxBytesReader(w, r.Body, capBytes)
	if err := r.ParseMultipartForm(capBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid upload (too large or malformed)")
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		files = r.MultipartForm.File["file"]
	}
	if len(files) != 1 {
		writeError(w, http.StatusBadRequest, "attach exactly one PDF or ZIP file")
		return
	}
	fh := files[0]

	if err := os.MkdirAll(uploadDir(), 0750); err != nil {
		writeError(w, http.StatusInternalServerError, "upload storage unavailable")
		return
	}
	stored, _, err := saveUploadedFile(fh, resMaxMB)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.q.CreateResource(r.Context(), db.CreateResourceParams{
		Title:      title,
		Fileurl:    "pending",
		Categoryid: categoryID,
	})
	if err != nil {
		os.Remove(filepath.Join(uploadDir(), stored))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := res.LastInsertId()
	servePath := fmt.Sprintf("/api/resources/%d/file", newID)
	if err := h.q.UpdateResourceFileURL(r.Context(), db.UpdateResourceFileURLParams{
		Fileurl:    servePath,
		Filename:   sql.NullString{String: stored, Valid: true},
		Resourceid: int32(newID),
	}); err != nil {
		os.Remove(filepath.Join(uploadDir(), stored))
		h.q.DeleteResource(r.Context(), int32(newID))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Optional batch scope: repeated "batchIds" fields or comma-separated.
	var batchIDs []int32
	seen := map[string]bool{}
	for _, raw := range r.MultipartForm.Value["batchIds"] {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			if n, err := strconv.ParseInt(part, 10, 32); err == nil {
				batchIDs = append(batchIDs, int32(n))
			}
		}
	}
	if err := h.setResourceBatches(r.Context(), int32(newID), batchIDs); err != nil {
		os.Remove(filepath.Join(uploadDir(), stored))
		h.q.DeleteResource(r.Context(), int32(newID))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message":    "resource uploaded",
		"resourceId": newID,
	})
}

// ResourceFile serves a file-backed resource to enrolled course members.
// Link rows (FileURL not served by us) 404 here; their URLs open directly.
func (h *AppHandler) ResourceFile(w http.ResponseWriter, r *http.Request) {
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
	if !h.requireCourseMember(w, r, resource.Courseid) {
		return
	}
	if !resource.Filename.Valid || resource.Filename.String == "" {
		writeError(w, http.StatusNotFound, "this resource is a link, not a file")
		return
	}
	name := filepath.Base(resource.Filename.String)
	if name != resource.Filename.String || strings.Contains(name, "..") {
		writeError(w, http.StatusNotFound, "resource file not found")
		return
	}

	f, err := os.Open(filepath.Join(uploadDir(), name))
	if err != nil {
		writeError(w, http.StatusNotFound, "resource file not found")
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusNotFound, "resource file not found")
		return
	}

	ext := strings.ToLower(filepath.Ext(name))
	mime := "application/octet-stream"
	if m, ok := uploadExtMIME[ext]; ok {
		mime = m
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", resource.Title+ext))
	http.ServeContent(w, r, name, stat.ModTime(), f)
}
