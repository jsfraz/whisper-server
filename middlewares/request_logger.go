package middlewares

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDHeader is accepted from clients and echoed on every response.
const RequestIDHeader = "X-Request-ID"

func sanitizeForLog(v string) string {
	v = strings.ReplaceAll(v, "\n", "")
	v = strings.ReplaceAll(v, "\r", "")
	return v
}

// RequestLogger replaces Gin's access logger with structured slog output and
// attaches a request id that correlates every log line for one request.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// The container health check runs every 30s; keep it out of the log stream.
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		requestID := sanitizeForLog(c.GetHeader(RequestIDHeader))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set("requestId", requestID)
		c.Header(RequestIDHeader, requestID)

		start := time.Now()
		c.Next()

		attrs := []any{
			"request_id", requestID,
			"method", c.Request.Method,
			"path", sanitizeForLog(c.Request.URL.Path),
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", sanitizeForLog(c.ClientIP()),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "error", sanitizeForLog(c.Errors.String()))
		}
		slog.Info("request", attrs...)
	}
}
