package api_test

import (
	"encoding/binary"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/lost-woods/random/src/api"
	"github.com/lost-woods/random/src/rng"
)

type uint32CounterReader struct {
	next uint32
	buf  [4]byte
	off  int
}

func (r *uint32CounterReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		if r.off == 0 {
			binary.BigEndian.PutUint32(r.buf[:], r.next)
			r.next++
		}
		copied := copy(p[n:], r.buf[r.off:])
		n += copied
		r.off = (r.off + copied) % 4
	}
	return n, nil
}

var uuidV4Re = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestHandlers_AcceptHeaderControlsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rr := &uint32CounterReader{next: 1}
	health := rng.NewHealth()
	health.Set(true, "")

	// logger can be nil for these endpoints, but we'll provide a no-op to be safe
	zapLogger := zap.NewNop().Sugar()
	h := api.NewHandlers(rr, health, zapLogger)

	// JSON request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?min=1&max=3", nil)
	c.Request.Header.Set("Accept", "application/json")
	h.RandomNumber(c)

	if w.Code != 200 {
		t.Fatalf("json expected 200 got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "\"request_id\"") {
		t.Fatalf("json response missing request_id: %s", body)
	}

	// Extract request_id very simply (we avoid JSON decode because output shape might change slightly)
	rid := extractJSONField(body, "request_id")
	if rid == "" || !uuidV4Re.MatchString(rid) {
		t.Fatalf("invalid request_id: %q body=%s", rid, body)
	}

	// Plain text request (no Accept json)
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/?min=1&max=3", nil)
	h.RandomNumber(c2)

	if w2.Code != 200 {
		t.Fatalf("text expected 200 got %d: %s", w2.Code, w2.Body.String())
	}
	body2 := w2.Body.String()
	if !strings.Contains(body2, "request_id:") {
		t.Fatalf("text response missing request_id: %s", body2)
	}
}

func extractJSONField(body string, field string) string {
	// naive extractor for `"field":"value"`
	needle := `"` + field + `":"`
	i := strings.Index(body, needle)
	if i < 0 {
		return ""
	}
	start := i + len(needle)
	end := strings.Index(body[start:], `"`)
	if end < 0 {
		return ""
	}
	return body[start : start+end]
}
