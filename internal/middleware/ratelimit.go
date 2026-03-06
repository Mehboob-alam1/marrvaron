package middleware

import (
	"net/http"
	"strconv"
	"time"

	"marvaron/internal/database"

	"github.com/gin-gonic/gin"
)

// RateLimitMiddleware implementa rate limiting usando Redis
func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Usa IP address come chiave
		key := "ratelimit:" + c.ClientIP()

		// Conta richieste
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

		// Imposta header con informazioni rate limit
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(int(limit-count)))
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
