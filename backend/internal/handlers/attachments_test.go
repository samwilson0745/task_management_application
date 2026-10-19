package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createTask(t *testing.T, h http.Handler, token string) string {
	t.Helper()
	rec := doRequest(t, h, http.MethodPost, "/tasks/", token, map[string]string{"title": "With files"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create task: %d %s", rec.Code, rec.Body.String())
	}
	var task struct {
		ID string `json:"id"`
	}
	json.Unmarshal(rec.Body.Bytes(), &task)
	return task.ID
}

func upload(t *testing.T, h http.Handler, token, taskID, filename, claimedType string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	hdr := make(map[string][]string)
	hdr["Content-Disposition"] = []string{mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": filename})}
	hdr["Content-Type"] = []string{claimedType}
	part, err := mw.CreatePart(hdr)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(content)
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/tasks/"+taskID+"/attachments", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func uploadOK(t *testing.T, h http.Handler, token, taskID, filename, claimedType string, content []byte) string {
	t.Helper()
	rec := upload(t, h, token, taskID, filename, claimedType, content)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	var a struct {
		ID          string `json:"id"`
		ContentType string `json:"content_type"`
		SizeBytes   int64  `json:"size_bytes"`
	}
	json.Unmarshal(rec.Body.Bytes(), &a)
	if a.SizeBytes != int64(len(content)) {
		t.Errorf("size_bytes = %d, want %d", a.SizeBytes, len(content))
	}
	return a.ID
}

var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00")

func TestAttachmentStoredInDBAndServedSafely(t *testing.T) {
	h := setupTestServer(t)
	token := signupAndLogin(t, h, "erin@example.com")
	taskID := createTask(t, h, token)

	// A real PNG is previewable inline.
	pngID := uploadOK(t, h, token, taskID, "pic.png", "image/png", pngBytes)
	rec := doRequest(t, h, http.MethodGet, "/tasks/"+taskID+"/attachments/"+pngID, token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("download png: %d", rec.Code)
	}
	if !bytes.Equal(rec.Body.Bytes(), pngBytes) {
		t.Errorf("downloaded bytes differ from uploaded")
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("png Content-Type = %q", ct)
	}
	if d, _, _ := mime.ParseMediaType(rec.Header().Get("Content-Disposition")); d != "inline" {
		t.Errorf("png disposition = %q, want inline", d)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("missing nosniff header")
	}

	// HTML claiming to be an image is sniffed as HTML and forced to download.
	html := []byte(`<html><script>alert(1)</script></html>`)
	htmlID := uploadOK(t, h, token, taskID, `evil"; x=".png`, "image/png", html)
	rec = doRequest(t, h, http.MethodGet, "/tasks/"+taskID+"/attachments/"+htmlID, token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("download html: %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("html Content-Type = %q, want application/octet-stream", ct)
	}
	disp, params, err := mime.ParseMediaType(rec.Header().Get("Content-Disposition"))
	if err != nil || disp != "attachment" {
		t.Fatalf("html disposition = %q (err %v), want attachment", disp, err)
	}
	if params["filename"] != `evil"; x=".png` {
		t.Errorf("filename not round-tripped safely: %q", params["filename"])
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Errorf("missing CSP header")
	}

	// SVG must not be served inline either.
	svgID := uploadOK(t, h, token, taskID, "a.svg", "image/svg+xml", []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><script>1</script></svg>`))
	rec = doRequest(t, h, http.MethodGet, "/tasks/"+taskID+"/attachments/"+svgID, token, nil)
	if d, _, _ := mime.ParseMediaType(rec.Header().Get("Content-Disposition")); d != "attachment" {
		t.Errorf("svg disposition = %q, want attachment", d)
	}

	// Listing never leaks storage details.
	rec = doRequest(t, h, http.MethodGet, "/tasks/"+taskID+"/attachments", token, nil)
	var list struct {
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Data) != 3 {
		t.Fatalf("expected 3 attachments, got %d", len(list.Data))
	}
	for _, key := range []string{"storage_path", "data"} {
		if _, found := list.Data[0][key]; found {
			t.Errorf("list response exposes %q", key)
		}
	}

	// Deleting the task removes its attachments.
	doRequest(t, h, http.MethodDelete, "/tasks/"+taskID, token, nil)
	var n int
	testPool.QueryRow(context.Background(), `SELECT COUNT(*) FROM task_attachments`).Scan(&n)
	if n != 0 {
		t.Errorf("expected attachments removed with task, %d remain", n)
	}
}

func TestAttachmentAccessControl(t *testing.T) {
	h := setupTestServer(t)
	alice := signupAndLogin(t, h, "alice@example.com")
	bob := signupAndLogin(t, h, "bob@example.com")
	signupAndLogin(t, h, "root@example.com")
	if _, err := testPool.Exec(context.Background(), `UPDATE users SET role='admin' WHERE email='root@example.com'`); err != nil {
		t.Fatal(err)
	}
	// Log in again so the token carries the admin role.
	rec := doRequest(t, h, http.MethodPost, "/auth/login", "", map[string]string{"email": "root@example.com", "password": "password123"})
	var login struct {
		Token string `json:"token"`
	}
	json.Unmarshal(rec.Body.Bytes(), &login)
	admin := login.Token

	taskID := createTask(t, h, alice)
	attID := uploadOK(t, h, alice, taskID, "notes.txt", "text/plain", []byte("hello"))
	path := "/tasks/" + taskID + "/attachments"

	// Other users get 404 for everything.
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, path}, {http.MethodGet, path + "/" + attID}, {http.MethodDelete, path + "/" + attID},
	} {
		if rec := doRequest(t, h, c.method, c.path, bob, nil); rec.Code != http.StatusNotFound {
			t.Errorf("bob %s %s: %d, want 404", c.method, c.path, rec.Code)
		}
	}
	if rec := upload(t, h, bob, taskID, "x.txt", "text/plain", []byte("x")); rec.Code != http.StatusNotFound {
		t.Errorf("bob upload: %d, want 404", rec.Code)
	}

	// Admins can read but not modify.
	if rec := doRequest(t, h, http.MethodGet, path, admin, nil); rec.Code != http.StatusOK {
		t.Errorf("admin list: %d, want 200", rec.Code)
	}
	if rec := doRequest(t, h, http.MethodGet, path+"/"+attID, admin, nil); rec.Code != http.StatusOK {
		t.Errorf("admin download: %d, want 200", rec.Code)
	}
	if rec := upload(t, h, admin, taskID, "x.txt", "text/plain", []byte("x")); rec.Code != http.StatusNotFound {
		t.Errorf("admin upload: %d, want 404", rec.Code)
	}
	if rec := doRequest(t, h, http.MethodDelete, path+"/"+attID, admin, nil); rec.Code != http.StatusNotFound {
		t.Errorf("admin delete: %d, want 404", rec.Code)
	}

	// Oversized files are rejected.
	if rec := upload(t, h, alice, taskID, "big.bin", "application/octet-stream", make([]byte, 10<<20+1)); rec.Code != http.StatusBadRequest {
		t.Errorf("oversized upload: %d, want 400", rec.Code)
	}

	// Legacy rows with no stored contents report 404 rather than erroring.
	var legacyID string
	testPool.QueryRow(context.Background(), `
		INSERT INTO task_attachments (task_id, user_id, filename, content_type, size_bytes, storage_path)
		SELECT $1, user_id, 'old.txt', 'text/plain', 3, '/gone' FROM tasks WHERE id = $1
		RETURNING id`, taskID).Scan(&legacyID)
	if rec := doRequest(t, h, http.MethodGet, path+"/"+legacyID, alice, nil); rec.Code != http.StatusNotFound {
		t.Errorf("legacy download: %d, want 404", rec.Code)
	}

	// Owner can delete.
	if rec := doRequest(t, h, http.MethodDelete, path+"/"+attID, alice, nil); rec.Code != http.StatusNoContent {
		t.Errorf("owner delete: %d, want 204", rec.Code)
	}
}

func TestTicketAuth(t *testing.T) {
	h := setupTestServer(t)
	session := signupAndLogin(t, h, "frank@example.com")

	rec := doRequest(t, h, http.MethodPost, "/auth/ticket", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("ticket without auth: %d, want 401", rec.Code)
	}
	rec = doRequest(t, h, http.MethodPost, "/auth/ticket", session, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("ticket: %d %s", rec.Code, rec.Body.String())
	}
	var tr struct {
		Ticket string `json:"ticket"`
	}
	json.Unmarshal(rec.Body.Bytes(), &tr)

	get := func(path, bearer string) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		// An already-cancelled context makes the stream handler return right
		// after the initial ": connected" frame.
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req.WithContext(ctx))
		return w.Code
	}

	if c := get("/tasks/stream?ticket="+tr.Ticket, ""); c != http.StatusOK {
		t.Errorf("stream with ticket: %d, want 200", c)
	}
	if c := get("/tasks/stream?ticket="+session, ""); c != http.StatusUnauthorized {
		t.Errorf("stream with session JWT as ticket: %d, want 401", c)
	}
	if c := get("/tasks/stream?token="+session, ""); c != http.StatusUnauthorized {
		t.Errorf("stream with ?token=: %d, want 401", c)
	}
	if c := get("/tasks/stream", session); c != http.StatusOK {
		t.Errorf("stream with bearer: %d, want 200", c)
	}
	// Tickets and URL tokens work nowhere else.
	if c := get("/tasks/?ticket="+tr.Ticket, ""); c != http.StatusUnauthorized {
		t.Errorf("list with ?ticket=: %d, want 401", c)
	}
	if c := get("/tasks/?token="+session, ""); c != http.StatusUnauthorized {
		t.Errorf("list with ?token=: %d, want 401", c)
	}
	if c := get("/tasks/", tr.Ticket); c != http.StatusUnauthorized {
		t.Errorf("ticket as bearer: %d, want 401", c)
	}
}
