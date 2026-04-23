package middleware

import (
	"log"
	"time"

	"github.com/labstack/echo/v4"
)

// Timing は各リクエストの処理時間を計測してログ出力する Echo middleware を返す。
func Timing() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			elapsed := time.Since(start)
			log.Printf("[timing] %v %v took %v", c.Request().Method, c.Request().RequestURI, elapsed)

			return err
		}
	}
}
