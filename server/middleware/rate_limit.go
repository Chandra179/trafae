package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	// rateLimitIdlePrune removes client buckets idle longer than this.
	rateLimitIdlePrune = 10 * time.Minute
	// rateLimitMaxClients bounds memory: once the map reaches this size the
	// next new client triggers a prune of idle buckets.
	rateLimitMaxClients = 4096
)

type rateClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit returns middleware allowing roughly rps requests per second per
// client address, with bursts up to burst. Buckets are per RemoteAddr: the
// router runs with trusted proxies disabled, so a spoofed X-Forwarded-For
// cannot rotate limits or grow the map.
func RateLimit(rps float64, burst int) gin.HandlerFunc {
	var (
		mu      sync.Mutex
		clients = make(map[string]*rateClient)
	)
	return func(c *gin.Context) {
		key := c.ClientIP()
		mu.Lock()
		client, ok := clients[key]
		if !ok {
			if len(clients) >= rateLimitMaxClients {
				pruneRateClients(clients, time.Now())
			}
			client = &rateClient{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
			clients[key] = client
		}
		client.lastSeen = time.Now()
		mu.Unlock()

		if !client.limiter.Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func pruneRateClients(clients map[string]*rateClient, now time.Time) {
	for key, client := range clients {
		if now.Sub(client.lastSeen) > rateLimitIdlePrune {
			delete(clients, key)
		}
	}
}
