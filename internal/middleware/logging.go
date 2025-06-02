// internal/middleware/logging.go
package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func StructuredLogging() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		log := logrus.WithFields(logrus.Fields{
			"client_ip":   param.ClientIP,
			"timestamp":   param.TimeStamp.Format(time.RFC3339),
			"method":      param.Method,
			"path":        param.Path,
			"protocol":    param.Request.Proto,
			"status_code": param.StatusCode,
			"latency":     param.Latency,
			"user_agent":  param.Request.UserAgent(),
			"error":       param.ErrorMessage,
		})

		if param.StatusCode >= 400 {
			log.Error("HTTP request")
		} else {
			log.Info("HTTP request")
		}

		return ""
	})
}

func RequestLogging() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for health checks
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Read request body for logging (only for POST/PUT/PATCH)
		var requestBody []byte
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			if c.Request.Body != nil {
				requestBody, _ = io.ReadAll(c.Request.Body)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}
		}

		// Create response body writer
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		if raw != "" {
			path = path + "?" + raw
		}

		// Log the request
		logFields := logrus.Fields{
			"client_ip":    c.ClientIP(),
			"method":       c.Request.Method,
			"path":         path,
			"status_code":  c.Writer.Status(),
			"latency":      latency,
			"user_agent":   c.Request.UserAgent(),
			"request_size": c.Request.ContentLength,
		}

		// Add user ID if authenticated
		if userID, exists := c.Get("user_id"); exists {
			logFields["user_id"] = userID
		}

		// Add request body for non-sensitive endpoints
		if len(requestBody) > 0 && !isSensitiveEndpoint(path) {
			logFields["request_body"] = string(requestBody)
		}

		// Add error if exists
		if len(c.Errors) > 0 {
			logFields["errors"] = c.Errors.String()
		}

		logger := logrus.WithFields(logFields)

		if c.Writer.Status() >= 500 {
			logger.Error("Server error")
		} else if c.Writer.Status() >= 400 {
			logger.Warn("Client error")
		} else {
			logger.Info("Request processed")
		}
	}
}

func isSensitiveEndpoint(path string) bool {
	sensitiveEndpoints := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/user/change-password",
	}

	for _, endpoint := range sensitiveEndpoints {
		if path == endpoint {
			return true
		}
	}
	return false
}
