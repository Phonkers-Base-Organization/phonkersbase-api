package middlewares

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	RequestsPerMinute = 50
)

func RateLimitRequests() gin.HandlerFunc {
	var (
		limiter = make(map[string]*rate.Limiter)
		mu      = sync.Mutex{}
	)
	go Cleanup(limiter, &mu)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		if _, exists := limiter[ip]; !exists {
			limiter[ip] = rate.NewLimiter(rate.Every(time.Minute/time.Duration(RequestsPerMinute)), RequestsPerMinute)
		}
		
		if !limiter[ip].Allow() {
			c.AbortWithStatus(429)
			mu.Unlock()
			return
		}
		mu.Unlock()

		c.Next()
	}
}

func Cleanup(rlMap map[string]*rate.Limiter, mu *sync.Mutex) {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		mu.Lock()
		for ip, limiter := range rlMap {
			// Remove limiters that haven't been used recently
			
			if limiter.Tokens() >= float64(RequestsPerMinute) {
				delete(rlMap, ip)
			}
		}
		mu.Unlock()
	}
}
