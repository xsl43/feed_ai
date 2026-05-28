package admin

import (
	"feedsystem_ai_go/internal/middleware/jwt"
	"net/http"

	"github.com/gin-gonic/gin"
)

var adminIDs = map[uint]bool{1: true}

// SetAdminIDs sets the list of admin user IDs.
func SetAdminIDs(ids []uint) {
	adminIDs = make(map[uint]bool)
	for _, id := range ids {
		adminIDs[id] = true
	}
}

// IsAdmin checks whether the given account ID is an admin.
func IsAdmin(accountID uint) bool {
	return adminIDs[accountID]
}

// RequireAdmin is a Gin middleware that checks if the authenticated user is an admin.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, err := jwt.GetAccountID(c)
		if err != nil || !adminIDs[accountID] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			return
		}
		c.Next()
	}
}
