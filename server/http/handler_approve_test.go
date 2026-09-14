package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/teexue/nexakit/loop"
)

func TestHandleApprove(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	// Start a pending approval directly on the server's approver.
	approvalID := "appr-test-1"
	go func() {
		// Block briefly to ensure the registration happens before we resolve.
		srv.approver.Approve(context.Background(), loop.ApprovalRequest{
			Tool:       "test_tool",
			ApprovalID: approvalID,
		})
	}()

	// Wait for the goroutine to register the pending approval.
	time.Sleep(50 * time.Millisecond)

	body, _ := json.Marshal(ApproveRequest{
		ApprovalID: approvalID,
		Approved:   true,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["resolved"] != true {
		t.Fatalf("expected resolved=true, got %v", resp["resolved"])
	}
}

func TestHandleApprove_MissingID(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	body, _ := json.Marshal(ApproveRequest{Approved: true})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestHandleApprove_NotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	router := srv.Handler()

	body, _ := json.Marshal(ApproveRequest{ApprovalID: "does-not-exist", Approved: true})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/agents/approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}
