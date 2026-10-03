package router

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Chandra179/trafae/server/middleware"
)

// New builds the Gin engine: middleware chain, health routes, and module
// handlers.
func (d *dependencies) New() *gin.Engine {
	r := gin.New()

	middlewareChain := []gin.HandlerFunc{
		gin.CustomRecovery(func(c *gin.Context, err any) {
			path := c.FullPath()
			if path == "" {
				path = "<unmatched>"
			}
			fields := []zap.Field{
				zap.String("panic", fmt.Sprintf("%v", err)),
				zap.String("stack", string(debug.Stack())),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
			}
			if requestID := middleware.GetRequestID(c.Request.Context()); requestID != "" {
				fields = append(fields, zap.String("request_id", requestID))
			}
			d.logger.Error("panic recovered", fields...)
			c.AbortWithStatus(http.StatusInternalServerError)
		}),
		middleware.RequestID,
	}
	if d.requestBodyLimit != nil {
		middlewareChain = append(middlewareChain, d.requestBodyLimit)
	}
	if d.requestLog != nil {
		middlewareChain = append(middlewareChain, d.requestLog)
	}
	r.Use(middlewareChain...)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	if d.readiness != nil {
		r.GET("/ready", d.readiness)
	}
	r.POST("/example", d.example)
	if d.bookSearch != nil {
		r.GET("/books/search", d.bookSearch)
	}
	if d.bookProviders != nil {
		r.GET("/books/providers", d.bookProviders)
	}

	return r
}
