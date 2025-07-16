package api

import (
	//"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"wled-simulator/internal/state"

	"github.com/gin-gonic/gin"
)

type testState struct {
	On   bool `json:"on"`
	Bri  int  `json:"bri"`
	Live bool `json:"live"`
}

type testInfo struct {
	Ver  string `json:"ver"`
	Name string `json:"name"`
	Live bool   `json:"live"`
}

type testCombined struct {
	State testState `json:"state"`
	Info  testInfo  `json:"info"`
}

func TestGetState(t *testing.T) {
	ledState := state.NewLEDState(10, "#000000")
	srv := NewServer(":0", ledState)

	r := gin.Default()
	r.GET("/json/state", srv.handleGetState)

	req := httptest.NewRequest(http.MethodGet, "/json/state", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp testState
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if !resp.On {
		t.Fatalf("expected power on by default")
	}
	// Live should be false initially
	if resp.Live {
		t.Fatalf("expected live to be false initially")
	}
}

func TestGetInfo(t *testing.T) {
	ledState := state.NewLEDState(10, "#000000")
	srv := NewServer(":0", ledState)

	r := gin.Default()
	r.GET("/json/info", srv.handleGetInfo)

	req := httptest.NewRequest(http.MethodGet, "/json/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp testInfo
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}

	if resp.Ver != "simulator" {
		t.Fatalf("expected version 'simulator', got %s", resp.Ver)
	}
	if resp.Name != "WLED Simulator" {
		t.Fatalf("expected name 'WLED Simulator', got %s", resp.Name)
	}
	// Live should be false initially
	if resp.Live {
		t.Fatalf("expected live to be false initially")
	}
}

func TestGetJSON(t *testing.T) {
	ledState := state.NewLEDState(10, "#000000")
	srv := NewServer(":0", ledState)

	r := gin.Default()
	r.GET("/json", srv.handleGetJSON)

	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp testCombined
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}

	// Check state section
	if !resp.State.On {
		t.Fatalf("expected power on by default")
	}
	if resp.State.Live {
		t.Fatalf("expected state.live to be false initially")
	}

	// Check info section
	if resp.Info.Ver != "simulator" {
		t.Fatalf("expected version 'simulator', got %s", resp.Info.Ver)
	}
	if resp.Info.Live {
		t.Fatalf("expected info.live to be false initially")
	}
}

func TestLiveFieldWithDDPActivity(t *testing.T) {
	ledState := state.NewLEDState(10, "#000000")
	srv := NewServer(":0", ledState)

	r := gin.Default()
	r.GET("/json/info", srv.handleGetInfo)

	// Simulate DDP activity
	ledState.SetLive()

	req := httptest.NewRequest(http.MethodGet, "/json/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp testInfo
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}

	// Live should be true after SetLive()
	if !resp.Live {
		t.Fatalf("expected live to be true after DDP activity")
	}
}

func TestPortCollision(t *testing.T) {
	// Use a specific port for testing
	const testPort = ":8081"
	ledState := state.NewLEDState(10, "#000000")

	// Start first server
	srv1 := NewServer(testPort, ledState)
	errChan1 := make(chan error, 1)
	go func() {
		err := srv1.Start()
		errChan1 <- err // Always send the error, even if nil
	}()

	// Wait for first server to start
	select {
	case err := <-errChan1:
		if err != nil {
			t.Fatalf("First server failed unexpectedly: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		// Server started successfully (no error within timeout)
	}

	// Try to start second server on same port
	srv2 := NewServer(testPort, ledState)
	errChan2 := make(chan error, 1)
	go func() {
		err := srv2.Start()
		errChan2 <- err // Always send the error, even if nil
	}()

	// Wait for error from second server
	select {
	case err := <-errChan2:
		if err == nil {
			t.Fatal("Expected error when starting server on occupied port")
		}
		expectedErrMsg := "bind: address already in use"
		if !strings.Contains(err.Error(), expectedErrMsg) {
			t.Errorf("Expected error containing '%s', got: %v", expectedErrMsg, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for port collision error")
	}

	// Cleanup
	srv1.Stop()
	srv2.Stop()
}

func TestNoRouteHandler(t *testing.T) {
	// Use a specific port for testing
	const testPort = ":8082"
	ledState := state.NewLEDState(10, "#000000")

	// Start server
	srv := NewServer(testPort, ledState)
	errChan := make(chan error, 1)
	go func() {
		err := srv.Start()
		errChan <- err
	}()

	// Wait for server to start
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Server failed to start: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		// Server started successfully
	}

	// Test cases for non-existent routes
	tests := []struct {
		name           string
		path           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Non-existent JSON endpoint",
			path:           "/json/nonexistent",
			method:         "GET",
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"Not found"}`,
		},
		{
			name:           "Random path",
			path:           "/random/path",
			method:         "GET",
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"Not found"}`,
		},
		{
			name:           "POST to non-existent endpoint",
			path:           "/api/v1/test",
			method:         "POST",
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"Not found"}`,
		},
	}

	// Run tests
	client := &http.Client{}
	baseURL := "http://localhost" + testPort

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req, err := http.NewRequest(tt.method, baseURL+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Send request
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Check Content-Type header
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Expected Content-Type to contain application/json, got %s", contentType)
			}

			// Read and verify response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			// Trim any whitespace/newlines for comparison
			actualBody := strings.TrimSpace(string(body))
			if actualBody != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, actualBody)
			}

			// Verify activity was reported for JSON endpoints
			if strings.HasPrefix(tt.path, "/json/") {
				// Give a moment for activity to be processed
				time.Sleep(50 * time.Millisecond)
				// Could add method to check ledState's last activity if needed
			}
		})
	}

	// Cleanup
	if err := srv.Stop(); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
}
