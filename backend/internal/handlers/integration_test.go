package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"taskmanager/internal/db"
	"taskmanager/internal/handlers"
	"taskmanager/internal/migrations"
	"taskmanager/internal/repository"
	"taskmanager/internal/router"
)

// These tests require a running PostgreSQL instance. Set TEST_DATABASE_URL to run them, e.g.:
//   TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/taskmanager_test?sslmode=disable go test ./...
func setupTestServer(t *testing.T) http.Handler {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	// Start each test from a clean slate.
	if _, err := pool.Exec(ctx, "TRUNCATE tasks, users CASCADE"); err != nil {
		t.Fatalf("truncating tables: %v", err)
	}

	userRepo := repository.NewUserRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)

	authHandler := handlers.NewAuthHandler(userRepo, "test-secret")
	taskHandler := handlers.NewTaskHandler(taskRepo)

	return router.New(router.Deps{
		JWTSecret:     "test-secret",
		AllowedOrigin: "http://localhost:3000",
		AuthHandler:   authHandler,
		TaskHandler:   taskHandler,
	})
}

func doRequest(t *testing.T, h http.Handler, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&reqBody).Encode(body); err != nil {
			t.Fatalf("encoding request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func signupAndLogin(t *testing.T, h http.Handler, email string) string {
	t.Helper()

	rec := doRequest(t, h, http.MethodPost, "/auth/signup", "", map[string]string{
		"email":    email,
		"password": "password123",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup: expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding signup response: %v", err)
	}
	return resp.Token
}

func TestSignupLoginAndAuthFlow(t *testing.T) {
	h := setupTestServer(t)

	// Signup
	token := signupAndLogin(t, h, "alice@example.com")
	if token == "" {
		t.Fatal("expected non-empty token from signup")
	}

	// Duplicate signup should fail
	rec := doRequest(t, h, http.MethodPost, "/auth/signup", "", map[string]string{
		"email":    "alice@example.com",
		"password": "password123",
	})
	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409 for duplicate signup, got %d", rec.Code)
	}

	// Login with correct credentials
	rec = doRequest(t, h, http.MethodPost, "/auth/login", "", map[string]string{
		"email":    "alice@example.com",
		"password": "password123",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for login, got %d: %s", rec.Code, rec.Body.String())
	}

	// Login with wrong password
	rec = doRequest(t, h, http.MethodPost, "/auth/login", "", map[string]string{
		"email":    "alice@example.com",
		"password": "wrongpassword",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for wrong password, got %d", rec.Code)
	}

	// Accessing tasks without a token should be rejected
	rec = doRequest(t, h, http.MethodGet, "/tasks/", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 without token, got %d", rec.Code)
	}

	// Accessing tasks with a valid token should succeed
	rec = doRequest(t, h, http.MethodGet, "/tasks/", token, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 with valid token, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTaskCRUDAndOwnership(t *testing.T) {
	h := setupTestServer(t)

	aliceToken := signupAndLogin(t, h, "alice@example.com")
	bobToken := signupAndLogin(t, h, "bob@example.com")

	// Validation error on missing title
	rec := doRequest(t, h, http.MethodPost, "/tasks/", aliceToken, map[string]interface{}{
		"description": "missing title",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422 for missing title, got %d: %s", rec.Code, rec.Body.String())
	}

	// Create a task
	rec = doRequest(t, h, http.MethodPost, "/tasks/", aliceToken, map[string]interface{}{
		"title":       "Write report",
		"description": "Quarterly report",
		"status":      "todo",
		"priority":    "high",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 creating task, got %d: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding create response: %v", err)
	}
	if created.Title != "Write report" {
		t.Errorf("expected title 'Write report', got %q", created.Title)
	}

	// Bob cannot access Alice's task
	rec = doRequest(t, h, http.MethodGet, "/tasks/"+created.ID, bobToken, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 when another user accesses task, got %d", rec.Code)
	}

	// Alice can fetch her own task
	rec = doRequest(t, h, http.MethodGet, "/tasks/"+created.ID, aliceToken, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 fetching own task, got %d: %s", rec.Code, rec.Body.String())
	}

	// Update status to done
	rec = doRequest(t, h, http.MethodPatch, "/tasks/"+created.ID, aliceToken, map[string]interface{}{
		"status": "done",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 updating task, got %d: %s", rec.Code, rec.Body.String())
	}
	var updated struct {
		Status string `json:"status"`
	}
	json.Unmarshal(rec.Body.Bytes(), &updated)
	if updated.Status != "done" {
		t.Errorf("expected status 'done' after update, got %q", updated.Status)
	}

	// Bob cannot delete Alice's task
	rec = doRequest(t, h, http.MethodDelete, "/tasks/"+created.ID, bobToken, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 when another user deletes task, got %d", rec.Code)
	}

	// Alice deletes her own task
	rec = doRequest(t, h, http.MethodDelete, "/tasks/"+created.ID, aliceToken, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204 deleting own task, got %d", rec.Code)
	}

	// Task is gone
	rec = doRequest(t, h, http.MethodGet, "/tasks/"+created.ID, aliceToken, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 after delete, got %d", rec.Code)
	}
}

func TestTaskListFilterSearchSortPagination(t *testing.T) {
	h := setupTestServer(t)
	token := signupAndLogin(t, h, "carol@example.com")

	tasks := []map[string]interface{}{
		{"title": "Buy groceries", "status": "todo", "priority": "low"},
		{"title": "Write code", "status": "in_progress", "priority": "high"},
		{"title": "Review PR", "status": "todo", "priority": "medium"},
	}
	for _, task := range tasks {
		rec := doRequest(t, h, http.MethodPost, "/tasks/", token, task)
		if rec.Code != http.StatusCreated {
			t.Fatalf("creating task: expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	// Filter by status
	rec := doRequest(t, h, http.MethodGet, "/tasks/?status=todo", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var listResp struct {
		Data  []map[string]interface{} `json:"data"`
		Total int                      `json:"total"`
	}
	json.Unmarshal(rec.Body.Bytes(), &listResp)
	if listResp.Total != 2 {
		t.Errorf("expected 2 todo tasks, got %d", listResp.Total)
	}

	// Search by title
	rec = doRequest(t, h, http.MethodGet, "/tasks/?search=code", token, nil)
	json.Unmarshal(rec.Body.Bytes(), &listResp)
	if listResp.Total != 1 {
		t.Errorf("expected 1 task matching search 'code', got %d", listResp.Total)
	}

	// Pagination
	rec = doRequest(t, h, http.MethodGet, "/tasks/?page=1&page_size=2", token, nil)
	json.Unmarshal(rec.Body.Bytes(), &listResp)
	if len(listResp.Data) != 2 {
		t.Errorf("expected 2 tasks on page 1 with page_size=2, got %d", len(listResp.Data))
	}
	if listResp.Total != 3 {
		t.Errorf("expected total of 3, got %d", listResp.Total)
	}

	// Invalid status filter
	rec = doRequest(t, h, http.MethodGet, "/tasks/?status=bogus", token, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid status filter, got %d", rec.Code)
	}
}
