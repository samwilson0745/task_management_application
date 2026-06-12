package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"taskmanager/internal/httpx"
	"taskmanager/internal/middleware"
	"taskmanager/internal/repository"
)

const maxAttachmentSize = 10 << 20 // 10MB

type AttachmentHandler struct {
	tasks       *repository.TaskRepository
	attachments *repository.AttachmentRepository
	storageDir  string
}

func NewAttachmentHandler(tasks *repository.TaskRepository, attachments *repository.AttachmentRepository, storageDir string) *AttachmentHandler {
	return &AttachmentHandler{tasks: tasks, attachments: attachments, storageDir: storageDir}
}

// loadOwnedTask fetches a task and verifies the requesting user may access it.
func (h *AttachmentHandler) loadAccessibleTask(w http.ResponseWriter, r *http.Request, id string) (ok bool) {
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
	if !canAccess(task, userID, role) {
		httpx.Error(w, http.StatusNotFound, "task not found")
		return false
	}
	return true
}

func (h *AttachmentHandler) List(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if !h.loadAccessibleTask(w, r, taskID) {
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
	if !h.loadAccessibleTask(w, r, taskID) {
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

	if header.Size > maxAttachmentSize {
		httpx.Error(w, http.StatusBadRequest, "file exceeds 10MB limit")
		return
	}

	if err := os.MkdirAll(filepath.Join(h.storageDir, taskID), 0o755); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to store file")
		return
	}

	storedName, err := randomFilename()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to store file")
		return
	}
	storagePath := filepath.Join(h.storageDir, taskID, storedName)

	dst, err := os.Create(storagePath)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to store file")
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to store file")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	attachment, err := h.attachments.Create(r.Context(), repository.CreateAttachmentParams{
		TaskID:      taskID,
		UserID:      userID,
		Filename:    header.Filename,
		ContentType: contentType,
		SizeBytes:   written,
		StoragePath: storagePath,
	})
	if err != nil {
		os.Remove(storagePath)
		httpx.Error(w, http.StatusInternalServerError, "failed to save attachment")
		return
	}

	httpx.JSON(w, http.StatusCreated, attachment)
}

func (h *AttachmentHandler) Download(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if !h.loadAccessibleTask(w, r, taskID) {
		return
	}

	attachment, err := h.attachments.GetByID(r.Context(), chi.URLParam(r, "attachmentID"))
	if err != nil || attachment.TaskID != taskID {
		httpx.Error(w, http.StatusNotFound, "attachment not found")
		return
	}

	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+attachment.Filename+"\"")
	http.ServeFile(w, r, attachment.StoragePath)
}

func (h *AttachmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if !h.loadAccessibleTask(w, r, taskID) {
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
	os.Remove(attachment.StoragePath)

	w.WriteHeader(http.StatusNoContent)
}

func randomFilename() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
