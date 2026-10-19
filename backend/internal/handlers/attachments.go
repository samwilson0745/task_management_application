package handlers

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"taskmanager/internal/httpx"
	"taskmanager/internal/middleware"
	"taskmanager/internal/repository"
)

const maxAttachmentSize = 10 << 20 // 10MB

// inlineSafeTypes are the only content types served for in-browser preview.
// Anything else (HTML, SVG, XML, scripts...) is forced to download.
var inlineSafeTypes = map[string]bool{
	"image/png":       true,
	"image/jpeg":      true,
	"image/gif":       true,
	"image/webp":      true,
	"application/pdf": true,
	"text/plain":      true,
}

func baseType(contentType string) string {
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	return mt
}

type AttachmentHandler struct {
	tasks       *repository.TaskRepository
	attachments *repository.AttachmentRepository
}

func NewAttachmentHandler(tasks *repository.TaskRepository, attachments *repository.AttachmentRepository) *AttachmentHandler {
	return &AttachmentHandler{tasks: tasks, attachments: attachments}
}

// loadTask fetches the task and verifies access. Owners and admins may read;
// when write is true only the owner may proceed (admins are read-only).
func (h *AttachmentHandler) loadTask(w http.ResponseWriter, r *http.Request, id string, write bool) (ok bool) {
	userID := middleware.UserIDFromContext(r.Context())
	role := middleware.UserRoleFromContext(r.Context())

	task, err := h.tasks.GetByID(r.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			httpx.Error(w, http.StatusNotFound, "task not found")
			return false
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to fetch task")
		return false
	}
	allowed := canAccess(task, userID, role)
	if write {
		allowed = task.UserID == userID
	}
	if !allowed {
		httpx.Error(w, http.StatusNotFound, "task not found")
		return false
	}
	return true
}

func (h *AttachmentHandler) List(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if !h.loadTask(w, r, taskID, false) {
		return
	}

	attachments, err := h.attachments.ListByTask(r.Context(), taskID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list attachments")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": attachments})
}

func (h *AttachmentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	taskID := chi.URLParam(r, "id")
	if !h.loadTask(w, r, taskID, true) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentSize+1<<20)
	if err := r.ParseMultipartForm(maxAttachmentSize); err != nil {
		httpx.Error(w, http.StatusBadRequest, "file too large or invalid form (max 10MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxAttachmentSize+1))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "failed to read file")
		return
	}
	if len(data) > maxAttachmentSize {
		httpx.Error(w, http.StatusBadRequest, "file exceeds 10MB limit")
		return
	}

	filename := filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
	if filename == "." || filename == "/" || filename == "" {
		filename = "file"
	}
	if len(filename) > 255 {
		filename = filename[:255]
	}

	// Never trust the client-supplied Content-Type: sniff the real contents.
	contentType := http.DetectContentType(data)

	attachment, err := h.attachments.Create(r.Context(), repository.CreateAttachmentParams{
		TaskID:      taskID,
		UserID:      userID,
		Filename:    filename,
		ContentType: contentType,
		Data:        data,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to save attachment")
		return
	}

	httpx.JSON(w, http.StatusCreated, attachment)
}

func (h *AttachmentHandler) Download(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if !h.loadTask(w, r, taskID, false) {
		return
	}

	attachment, err := h.attachments.GetByID(r.Context(), chi.URLParam(r, "attachmentID"))
	if err != nil || attachment.TaskID != taskID {
		httpx.Error(w, http.StatusNotFound, "attachment not found")
		return
	}
	data, err := h.attachments.GetData(r.Context(), attachment.ID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "attachment contents are no longer available")
		return
	}

	disposition, contentType := "attachment", "application/octet-stream"
	if mt := baseType(attachment.ContentType); inlineSafeTypes[mt] {
		disposition, contentType = "inline", attachment.ContentType
	}

	hdr := w.Header()
	hdr.Set("Content-Type", contentType)
	hdr.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": attachment.Filename}))
	hdr.Set("X-Content-Type-Options", "nosniff")
	hdr.Set("Content-Security-Policy", "sandbox; default-src 'none'")
	hdr.Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(data))
}

func (h *AttachmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if !h.loadTask(w, r, taskID, true) {
		return
	}

	attachment, err := h.attachments.GetByID(r.Context(), chi.URLParam(r, "attachmentID"))
	if err != nil || attachment.TaskID != taskID {
		httpx.Error(w, http.StatusNotFound, "attachment not found")
		return
	}

	if err := h.attachments.Delete(r.Context(), attachment.ID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to delete attachment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
