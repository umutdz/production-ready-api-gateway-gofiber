package middleware

// import (
//     "time"
//     "fmt"

//     "github.com/gofiber/fiber/v2"
//     "api-gateway/pkg/metrics"
// )

// func PrometheusMiddleware() fiber.Handler {
//     return func(c *fiber.Ctx) error {
//         start := time.Now()
//         path := c.Path()
//         method := c.Method()

//         // İsteği işle
//         err := c.Next()

//         // İstek tamamlandıktan sonra metrikleri kaydet
//         status := c.Response().StatusCode()
//         duration := time.Since(start).Seconds()

//         statusStr := fmt.Sprintf("%d", status)
//         metrics.HttpRequestsTotal.WithLabelValues(path, method, statusStr).Inc()
//         metrics.HttpRequestDuration.WithLabelValues(path, method, statusStr).Observe(duration)

//         return err
//     }
// }

