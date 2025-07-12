package api

import (
	//"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wled-simulator/internal/state"

	"github.com/gin-gonic/gin"
)

type testState struct {
	On  bool `json:"on"`
	Bri int  `json:"bri"`
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
}
