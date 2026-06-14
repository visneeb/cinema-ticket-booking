package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for development; restrict in production.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS upgrades the connection to WebSocket and streams seat events for a showtime.
// @Summary      WebSocket seat-status stream
// @Tags         websocket
// @Param        showtime_id  path  string  true  "Showtime ID"
// @Router       /ws/showtimes/{showtime_id} [get]
func (h *Handler) ServeWS(c *gin.Context) {
	showtimeID := c.Param("showtime_id")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ServeWS] upgrade error: %v", err)
		return
	}
	log.Printf("[ServeWS] client connected showtime=%s", showtimeID)
	h.Hub.ServeClient(conn, showtimeID)
}