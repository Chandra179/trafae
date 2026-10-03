package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type contextKey string

const requestIDKey contextKey = "requestID"

const headerKey = "X-Request-ID"

const maxRequestIDLength = 128

func generateRequestID() string {
	return uuid.NewString()
}

func storeRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// RequestID is a Gin middleware that reads X-Request-ID from the request
// header, reusing it if present or generating a random one. The ID is stored
// in the request context and echoed in the response header.
func RequestID(c *gin.Context) {
	id := c.GetHeader(headerKey)
	if !validRequestID(id) {
		id = generateRequestID()
	}
	c.Header(headerKey, id)
	c.Request = c.Request.WithContext(storeRequestID(c.Request.Context(), id))
	c.Next()
}

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > maxRequestIDLength {
		return false
	}

	for i := 0; i < len(id); i++ {
		char := id[i]
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' || char == '.' || char == ':' {
			continue
		}
		return false
	}
	return true
}
