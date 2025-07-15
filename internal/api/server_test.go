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
