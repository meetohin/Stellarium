package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_duration_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"path", "method", "status"})

	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"path", "method", "status"})

	activeConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_active_connections",
		Help: "Number of active HTTP connections.",
	})
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		activeConnections.Inc()

		c.Next()

		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		method := c.Request.Method

		// Record metrics
		httpDuration.WithLabelValues(path, method, status).Observe(duration.Seconds())
		httpRequests.WithLabelValues(path, method, status).Inc()
		activeConnections.Dec()
	}
}
