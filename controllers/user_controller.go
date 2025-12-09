// controllers/user_controller.go
package controllers

import (
	"log"
	"my-ecomm/config"
	"my-ecomm/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserController struct{}

func NewUserController() *UserController {
	return &UserController{}
}

// GetAllUsers retrieves all users with optional search
func (uc *UserController) GetAllUsers(c *gin.Context) {
	currentUserID := c.GetUint("userID")
	searchQuery := c.Query("search")

	var users []models.User
	query := config.DB.Where("id != ?", currentUserID) // Exclude current user

	if searchQuery != "" {
		query = query.Where("name LIKE ? OR email LIKE ?", "%"+searchQuery+"%", "%"+searchQuery+"%")
	}

	if err := query.Select("id", "name", "email", "created_at", "updated_at").Find(&users).Error; err != nil {
		log.Printf("GetAllUsers error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// GetUserByID retrieves a specific user by ID
func (uc *UserController) GetUserByID(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := config.DB.Select("id", "name", "email", "created_at", "updated_at").First(&user, userID).Error; err != nil {
		log.Printf("GetUserByID error: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}
