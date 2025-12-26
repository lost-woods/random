package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type responder struct{ c *gin.Context }

func (r responder) wantsJSON() bool {
	accept := strings.ToLower(r.c.GetHeader("Accept"))
	return strings.Contains(accept, "application/json")
}

func (r responder) err(status int, msg string) {
	if r.wantsJSON() {
		r.c.JSON(status, gin.H{"error": msg})
		return
	}
	r.c.String(status, msg)
}

func (r responder) ok(text string, payload gin.H, requestID string) {
	if r.wantsJSON() {
		out := gin.H{"request_id": requestID}
		for k, v := range payload {
			out[k] = v
		}
		r.c.JSON(http.StatusOK, out)
		return
	}
	r.c.String(http.StatusOK, text+"\nrequest_id: "+requestID)
}
