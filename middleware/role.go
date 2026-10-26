package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {

		roleValue, exists := c.Get("role")

		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Role user tidak ditemukan",
			})
			c.Abort()
			return
		}

		role, ok := roleValue.(string)

		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Role user tidak valid",
			})
			c.Abort()
			return
		}

		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Anda tidak memiliki akses ke endpoint ini",
		})

		c.Abort()
	}
}
