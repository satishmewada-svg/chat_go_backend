package controllers

import (
	"my-ecomm/config"
	"my-ecomm/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserController struct{}

func NewUserController() *UserController {
	return &UserController{}
}

func (uc *UserController) GetAllUsers(c *gin.Context) {
	var users []models.User

	// Get all users except the current one
	currentUserID := c.GetUint("userID")

	if err := config.DB.
		Select("id, name, username, email, is_online, last_seen_at, created_at, updated_at").
		Where("id != ?", currentUserID).
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (uc *UserController) GetUserByID(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := config.DB.
		Select("id, name, username, email, is_online, last_seen_at, created_at, updated_at").
		First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}
