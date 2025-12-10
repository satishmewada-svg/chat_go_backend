package controllers

import (
	"my-ecomm/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PresenceController struct {
	presenceService *services.PresenceService
}

func NewPresenceController() *PresenceController {
	return &PresenceController{
		presenceService: services.GetPresenceService(),
	}
}

// Heartbeat endpoint - called by frontend every 30 seconds
func (pc *PresenceController) Heartbeat(c *gin.Context) {
	userID := c.GetUint("userID")

	pc.presenceService.Heartbeat(userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "heartbeat received",
	})
}

// GetOnlineStatus returns online status of specific users
func (pc *PresenceController) GetOnlineStatus(c *gin.Context) {
	var req struct {
		UserIDs []uint `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	onlineStatus := make(map[uint]bool)
	for _, userID := range req.UserIDs {
		onlineStatus[userID] = pc.presenceService.IsUserOnline(userID)
	}

	c.JSON(http.StatusOK, gin.H{
		"online_status": onlineStatus,
	})
}

// GetAllOnlineUsers returns list of all online users
func (pc *PresenceController) GetAllOnlineUsers(c *gin.Context) {
	onlineUserIDs := pc.presenceService.GetOnlineUsers()

	c.JSON(http.StatusOK, gin.H{
		"online_users": onlineUserIDs,
	})
}
