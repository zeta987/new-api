package controller

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func StreamLogEvents(c *gin.Context) {
	userId := c.GetInt("id")
	includeAll := c.GetInt("role") >= common.RoleAdminUser
	events, unsubscribe := model.SubscribeLogEvents(userId, includeAll)
	defer unsubscribe()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	if _, err := fmt.Fprint(c.Writer, "event: ready\ndata: {}\n\n"); err != nil {
		return
	}
	c.Writer.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-events:
			if _, err := fmt.Fprint(c.Writer, "event: log\ndata: {}\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}
