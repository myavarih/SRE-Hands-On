package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/myavarih/sre-hands-on/pkg/telemetry"
	"github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/controllers"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	limiters *expirable.LRU[string, *rate.Limiter]
	r        rate.Limit
	b        int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		// 10,000 max cached IP limiters with automatic 10-minute TTL expiration (Flaw F-18 fix)
		limiters: expirable.NewLRU[string, *rate.Limiter](10000, nil, 10*time.Minute),
		r:        r,
		b:        b,
	}
}

func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	limiter, exists := i.limiters.Get(ip)
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.limiters.Add(ip, limiter)
	}
	return limiter
}

// RateLimitMiddleware enforces token bucket rate limiting per client IP.
func RateLimitMiddleware(serviceName string, reqPerSecond float64, burst int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(rate.Limit(reqPerSecond), burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := limiter.getLimiter(ip)

		if !l.Allow() {
			telemetry.RateLimitRejections.WithLabelValues(serviceName, c.FullPath()).Inc()
			c.Header("Retry-After", "1")
			controllers.RespondError(c, http.StatusTooManyRequests, "Rate limit exceeded. Please slow down.", "")
			c.Abort()
			return
		}

		c.Next()
	}
}
