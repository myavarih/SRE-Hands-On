package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/myavarih/sre-hands-on/pkg/telemetry"
	"github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/controllers"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		r:   r,
		b:   b,
	}
}

func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
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
