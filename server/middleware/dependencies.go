package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type dependencies struct {
	logger *zap.Logger
}

func NewDependencies(logger *zap.Logger) *dependencies {
	return &dependencies{
		logger: logger,
	}
}

// RequestBodyLimit rejects request bodies larger than maxBytes while leaving
// response and request logging to the other middleware in the chain.
func RequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
