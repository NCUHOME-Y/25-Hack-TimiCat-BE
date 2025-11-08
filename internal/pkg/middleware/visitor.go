package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const VisitorCookie = "tcid"

func Visitor() gin.HandlerFunc {
	return func(c *gin.Context) {
		vid, err := c.Cookie(VisitorCookie)
		if err != nil || vid == "" {
			vid = uuid.NewString()
			// 开发环境：SameSite Lax，HttpOnly；上线走 Secure=true
			c.SetCookie(VisitorCookie, vid, 3600*24*365, "/", "", false, true)
		}
		c.Set("visitor_id", vid)
		c.Next()
	}
}
