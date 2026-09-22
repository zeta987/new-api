package middleware

import "github.com/gin-gonic/gin"

func setNoStore(c *gin.Context) {
	headers := c.Writer.Header()
	headers.Set("Cache-Control", "no-store")
	headers.Del("Cache-Version")
	headers.Del("Expires")
	headers.Del("Pragma")
}

// NoStore marks a response as non-storable before downstream middleware can abort.
func NoStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		setNoStore(c)
		c.Next()
	}
}

func DisableCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private, max-age=0")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}
