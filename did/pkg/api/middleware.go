package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// metricsMiddleware 监控指标中间件
func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 增加活跃连接数
		s.activeConnections.Inc()
		defer s.activeConnections.Dec()

		// 处理请求
		c.Next()

		// 记录指标
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		s.requestCounter.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
		s.requestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
	}
}

// rateLimitMiddleware 限流中间件
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 简单的限流实现，可以根据需要扩展
		c.Next()
	}
}
