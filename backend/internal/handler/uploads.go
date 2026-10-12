package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"moodleplusplus/internal/db"
)

// File uploads for PDF/ZIP assignments. Local disk storage (UPLOAD_DIR,
// default ./uploads); no S3. Served only through SubmissionFile below so
// every download passes requireSelfOrCourseStaff — never a naked FileServer
// (stored names are unguessable, but authz must not depend on that).

func uploadDir() string {
	if d := os.Getenv("UPLOAD_DIR"); d != "" {
		return d
	}
	return "uploads"
}

var uploadExtMIME = map[string]string{
	".pdf": "application/pdf",
	".zip": "application/zip",
}

func randHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// UploadSubmission accepts 1..MaxUploadFiles files (field "files" or "file").
// Each file becomes its own submission row (a set = same-second rows);
// resubmits are new rows, old files are kept as immutable history.
func (h *AppHandler) UploadSubmission(w http.ResponseWriter, r *http.Request) {
	assignmentID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid assignment id")
		return
	}

	assignment, err := h.q.GetAssignmentByID(r.Context(), assignmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "assignment not found")
		return
	}
	if !h.requireCourseStudent(w, r, assignment.Courseid) {
		return
	}

	visible, err := h.q.CountAssignmentVisibleToStudent(r.Context(), db.CountAssignmentVisibleToStudentParams{
		Assignmentid: assignmentID,
		Courseid:     assignment.Courseid,
		Userid:       h.callerID(r),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if visible == 0 {
		writeError(w, http.StatusForbidden, "assignment is not assigned to your batch")
		return
	}

	// Uploads are for file formats only; GitHub assignments use the JSON path.
	if assignment.Formatrequired.Valid {
		fmtReq := strings.ToUpper(assignment.Formatrequired.String)
		if strings.Contains(fmtReq, "GITHUB") || strings.Contains(fmtReq, "REPO") {
			writeError(w, http.StatusBadRequest, "this assignment takes a link submission, not a file upload")
			return
		}
	}

	maxMB := assignment.Maxuploadmb
	if maxMB <= 0 {
		maxMB = 25
	}
	maxFiles := assignment.Maxuploadfiles
	if maxFiles <= 0 {
		maxFiles = 1
	}
	capBytes := int64(maxMB)<<20*int64(maxFiles) + 1<<20
	r.Body = http.MaxBytesReader(w, r.Body, capBytes)
	if err := r.ParseMultipartForm(capBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid upload (too large or malformed)")
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		files = r.MultipartForm.File["file"]
	}
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "no file attached (field \"files\")")
		return
	}
	if int32(len(files)) > maxFiles {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("at most %d file(s) per submission", maxFiles))
		return
	}

	if err := os.MkdirAll(uploadDir(), 0750); err != nil {
		writeError(w, http.StatusInternalServerError, "upload storage unavailable")
		return
	}

	userID := h.callerID(r)
	submitted := make([]int32, 0, len(files))
	for _, fh := range files {
		if fh.Size > int64(maxMB)<<20 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("%s exceeds the %d MB limit", fh.Filename, maxMB))
			return
		}
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		wantMIME, ok := uploadExtMIME[ext]
		if !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("%s: only PDF and ZIP files are accepted", fh.Filename))
			return
		}
		src, err := fh.Open()
		if err != nil {
			writeError(w, http.StatusBadRequest, "cannot read uploaded file")
			return
		}
		head := make([]byte, 512)
		n, _ := io.ReadFull(src, head)
		if http.DetectContentType(head[:n]) != wantMIME {
			src.Close()
			writeError(w, http.StatusBadRequest, fmt.Sprintf("%s: content does not match its %s extension", fh.Filename, ext))
			return
		}
		if _, err := src.Seek(0, io.SeekStart); err != nil {
			src.Close()
			writeError(w, http.StatusInternalServerError, "cannot read uploaded file")
			return
		}

		tag, err := randHex(8)
		if err != nil {
			src.Close()
			writeError(w, http.StatusInternalServerError, "cannot store uploaded file")
			return
		}
		stored := tag + ext

		res, err := h.q.CreateSubmission(r.Context(), db.CreateSubmissionParams{
			Contentdata:  fh.Filename,
			Assignmentid: assignmentID,
			Userid:       userID,
			Filename:     sql.NullString{String: stored, Valid: true},
			Mimetype:     sql.NullString{String: wantMIME, Valid: true},
		})
		if err != nil {
			src.Close()
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		subID, _ := res.LastInsertId()

		dst, err := os.OpenFile(filepath.Join(uploadDir(), stored), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
		if err != nil {
			src.Close()
			h.q.DeleteSubmissionByID(r.Context(), int32(subID))
			writeError(w, http.StatusInternalServerError, "cannot store uploaded file")
			return
		}
		_, copyErr := io.Copy(dst, src)
		dst.Close()
		src.Close()
		if copyErr != nil {
			os.Remove(filepath.Join(uploadDir(), stored))
			h.q.DeleteSubmissionByID(r.Context(), int32(subID))
			writeError(w, http.StatusInternalServerError, "cannot store uploaded file")
			return
		}
		submitted = append(submitted, int32(subID))
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message":       "uploaded successfully",
		"submissionIds": submitted,
	})
}

// removeStoredFile deletes one upload by stored name. Rejects anything
// that isn't a bare filename (no traversal, paranoid by design).
func removeStoredFile(name string) error {
	if name == "" || name != filepath.Base(name) || strings.Contains(name, "..") {
		return fmt.Errorf("unsafe stored name")
	}
	return os.Remove(filepath.Join(uploadDir(), name))
}

func (h *AppHandler) SubmissionFile(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid submission id")
		return
	}

	sub, err := h.q.GetSubmissionByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "submission not found")
		return
	}
	if !h.requireSelfOrCourseStaff(w, r, sub.Userid, sub.Courseid) {
		return
	}
	if !sub.Filename.Valid || sub.Filename.String == "" {
		writeError(w, http.StatusNotFound, "this submission has no attached file")
		return
	}
	name := filepath.Base(sub.Filename.String)
	if name != sub.Filename.String || strings.Contains(name, "..") {
		writeError(w, http.StatusNotFound, "submission file not found")
		return
	}

	f, err := os.Open(filepath.Join(uploadDir(), name))
	if err != nil {
		writeError(w, http.StatusNotFound, "submission file not found")
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusNotFound, "submission file not found")
		return
	}

	mime := "application/octet-stream"
	if sub.Mimetype.Valid && sub.Mimetype.String != "" {
		mime = sub.Mimetype.String
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", sub.Contentdata))
	http.ServeContent(w, r, name, stat.ModTime(), f)
}
