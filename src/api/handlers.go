package api

import (
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/lost-woods/random/src/rng"
)

type Handlers struct {
	r      io.Reader
	health *rng.Health
	log    *zap.SugaredLogger
}

func NewHandlers(r io.Reader, h *rng.Health, log *zap.SugaredLogger) *Handlers {
	return &Handlers{r: r, health: h, log: log}
}

func (h *Handlers) rngOK(c *gin.Context) bool {
	if h.health == nil {
		responder{c}.err(http.StatusServiceUnavailable, "RNG unhealthy: missing health monitor")
		return false
	}

	ok, msg, _ := h.health.Snapshot()
	if ok {
		return true
	}

	responder{c}.err(http.StatusServiceUnavailable, "RNG unhealthy: "+msg)
	return false
}

func (h *Handlers) uuidFromRNG() (string, error) {
	id, err := rng.NewUUIDv4FromRNG(h.r)
	if err != nil && h.health != nil {
		h.health.Set(false, "error fetching random bytes for uuid: "+err.Error())
	}
	return id, err
}

/*
handleRNG enforces:
1. RNG health check
2. Outcome computation (NO UUID here)
3. Error handling
4. UUID generation ONLY after success
5. JSON vs plaintext response
*/
func (h *Handlers) handleRNG(
	c *gin.Context,
	work func() (text string, payload gin.H, status int, errMsg string),
) {
	if !h.rngOK(c) {
		return
	}

	text, payload, status, errMsg := work()
	if errMsg != "" {
		responder{c}.err(status, errMsg)
		return
	}

	requestID, err := h.uuidFromRNG()
	if err != nil {
		responder{c}.err(http.StatusInternalServerError, "Error generating request id.")
		return
	}

	responder{c}.ok(text, payload, requestID)
}

func APIKeyFromEnv() string { return os.Getenv("API_KEY") }

func CheckHeader(headerName, expectedValue string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Auth disabled if not configured
		if expectedValue == "" {
			c.Next()
			return
		}

		if c.GetHeader(headerName) != expectedValue {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
