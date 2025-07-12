package api

import (
	"context"
	"net/http"

	"wled-simulator/internal/state"

	"github.com/gin-gonic/gin"
)

type Server struct {
	addr   string
	state  *state.LEDState
	server *http.Server
}

func NewServer(addr string, s *state.LEDState) *Server {
	return &Server{addr: addr, state: s}
}

func (s *Server) Start() error {
	r := gin.Default()

	r.GET("/json", s.handleGetJSON)
	r.GET("/json/state", s.handleGetState)
	r.GET("/json/info", s.handleGetInfo)
	r.POST("/json/state", s.handlePostState)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: r,
	}

	return s.server.ListenAndServe()
}

func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Shutdown(context.Background())
	}
	return nil
}

type statePayload struct {
	On  *bool        `json:"on,omitempty"`
	Bri *int         `json:"bri,omitempty"`
	Seg []segPayload `json:"seg,omitempty"`
}

type segPayload struct {
	Col [][]int `json:"col,omitempty"`
}

func (s *Server) handleGetJSON(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"state": gin.H{
			"on":  s.state.Power(),
			"bri": s.state.Brightness(),
		},
		"info": gin.H{
			"ver": "simulator",
		},
	})
}

func (s *Server) handleGetState(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"on":  s.state.Power(),
		"bri": s.state.Brightness(),
	})
}

func (s *Server) handleGetInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ver":  "simulator",
		"name": "WLED Simulator",
	})
}

func (s *Server) handlePostState(c *gin.Context) {
	var p statePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if p.On != nil {
		s.state.SetPower(*p.On)
	}
	if p.Bri != nil {
		s.state.SetBrightness(*p.Bri)
	}
	c.Status(http.StatusNoContent)
}
