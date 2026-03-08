package middleware

import (
	"net/http"
	"strconv"
	"time"

	"marvaron/internal/database"

	"github.com/gin-gonic/gin"
)

// RateLimitMiddleware implements rate limiting using Redis.
// If Redis is not connected, requests are allowed (no rate limiting).
func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if database.RedisClient == nil {
			c.Next()
			return
		}

		key := "ratelimit:" + c.ClientIP()

		count, err := database.RedisClient.Incr(c.Request.Context(), key).Result()
		if err != nil {
			// Se Redis non è disponibile, continua senza rate limiting
			c.Next()
			return
		}

		// Imposta TTL alla prima richiesta
		if count == 1 {
			database.RedisClient.Expire(c.Request.Context(), key, window)
		}

		remaining := limit - int(count)
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(window).Unix(), 10))

		// Verifica limite
		if count > int64(limit) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
