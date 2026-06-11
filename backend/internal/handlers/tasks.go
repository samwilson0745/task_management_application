package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"taskmanager/internal/httpx"
	"taskmanager/internal/middleware"
	"taskmanager/internal/models"
	"taskmanager/internal/repository"
	"taskmanager/internal/validation"
)

type TaskHandler struct {
	tasks *repository.TaskRepository
}

func NewTaskHandler(tasks *repository.TaskRepository) *TaskHandler {
	return &TaskHandler{tasks: tasks}
}

type taskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	DueDate     *string `json:"due_date"`
}

type taskListResponse struct {
	Data       []models.Task `json:"data"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	Total      int           `json:"total"`
	TotalPages int           `json:"total_pages"`
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	status := q.Get("status")
	if status != "" && !models.TaskStatus(status).Valid() {
		httpx.Error(w, http.StatusBadRequest, "invalid status filter")
		return
	}

	sortBy := q.Get("sort_by")
	if sortBy != "" {
		switch sortBy {
		case "due_date", "priority", "created_at":
		default:
			httpx.Error(w, http.StatusBadRequest, "invalid sort_by; must be one of due_date, priority, created_at")
			return
		}
	}

	sortDir := q.Get("sort_dir")
	if sortDir != "" && sortDir != "asc" && sortDir != "desc" {
		httpx.Error(w, http.StatusBadRequest, "invalid sort_dir; must be asc or desc")
		return
	}

	filter := repository.TaskFilter{
		UserID:   userID,
		Status:   status,
		Search:   q.Get("search"),
		SortBy:   sortBy,
		SortDir:  sortDir,
		Page:     page,
		PageSize: pageSize,
	}

	tasks, total, err := h.tasks.List(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + filter.PageSize - 1) / filter.PageSize
	}

	httpx.JSON(w, http.StatusOK, taskListResponse{
		Data:       tasks,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	task, err := h.tasks.GetByID(r.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			httpx.Error(w, http.StatusNotFound, "task not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to fetch task")
		return
	}

	if task.UserID != userID {
		httpx.Error(w, http.StatusNotFound, "task not found")
		return
	}

	httpx.JSON(w, http.StatusOK, task)
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var req taskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	errs := validation.Errors{}
	if validation.IsBlank(req.Title) {
		errs.Add("title", "is required")
	}
	if len(req.Title) > 200 {
		errs.Add("title", "must be 200 characters or fewer")
	}

	status := models.TaskStatus(req.Status)
	if req.Status == "" {
		status = models.StatusTodo
	} else if !status.Valid() {
		errs.Add("status", "must be one of todo, in_progress, done")
	}

	priority := models.TaskPriority(req.Priority)
	if req.Priority == "" {
		priority = models.PriorityMedium
	} else if !priority.Valid() {
		errs.Add("priority", "must be one of low, medium, high")
	}

	if req.DueDate != nil && *req.DueDate != "" {
		if _, err := time.Parse(time.RFC3339, *req.DueDate); err != nil {
			errs.Add("due_date", "must be a valid RFC3339 timestamp")
		}
	}

	if errs.HasErrors() {
		httpx.ValidationError(w, errs)
		return
	}

	var dueDate *string
	if req.DueDate != nil && *req.DueDate != "" {
		dueDate = req.DueDate
	}

	task, err := h.tasks.Create(r.Context(), repository.CreateTaskParams{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		Priority:    priority,
		DueDate:     dueDate,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	httpx.JSON(w, http.StatusCreated, task)
}

type taskUpdateRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	DueDate     *string `json:"due_date"`
	ClearDue    bool    `json:"clear_due_date"`
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	existing, err := h.tasks.GetByID(r.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			httpx.Error(w, http.StatusNotFound, "task not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to fetch task")
		return
	}
	if existing.UserID != userID {
		httpx.Error(w, http.StatusNotFound, "task not found")
		return
	}

	var req taskUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	errs := validation.Errors{}
	params := repository.UpdateTaskParams{}

	if req.Title != nil {
		if validation.IsBlank(*req.Title) {
			errs.Add("title", "is required")
		} else if len(*req.Title) > 200 {
			errs.Add("title", "must be 200 characters or fewer")
		} else {
			params.Title = req.Title
		}
	}

	if req.Description != nil {
		params.Description = req.Description
	}

	if req.Status != nil {
		status := models.TaskStatus(*req.Status)
		if !status.Valid() {
			errs.Add("status", "must be one of todo, in_progress, done")
		} else {
			params.Status = &status
		}
	}

	if req.Priority != nil {
		priority := models.TaskPriority(*req.Priority)
		if !priority.Valid() {
			errs.Add("priority", "must be one of low, medium, high")
		} else {
			params.Priority = &priority
		}
	}

	if req.ClearDue {
		var nilStr *string
		params.DueDate = &nilStr
	} else if req.DueDate != nil {
		if *req.DueDate == "" {
			var nilStr *string
			params.DueDate = &nilStr
		} else if _, err := time.Parse(time.RFC3339, *req.DueDate); err != nil {
			errs.Add("due_date", "must be a valid RFC3339 timestamp")
		} else {
			due := req.DueDate
			params.DueDate = &due
		}
	}

	if errs.HasErrors() {
		httpx.ValidationError(w, errs)
		return
	}

	task, err := h.tasks.Update(r.Context(), id, params)
	if err != nil {
		if err == repository.ErrNotFound {
			httpx.Error(w, http.StatusNotFound, "task not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	httpx.JSON(w, http.StatusOK, task)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	existing, err := h.tasks.GetByID(r.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			httpx.Error(w, http.StatusNotFound, "task not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to fetch task")
		return
	}
	if existing.UserID != userID {
		httpx.Error(w, http.StatusNotFound, "task not found")
		return
	}

	if err := h.tasks.Delete(r.Context(), id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
