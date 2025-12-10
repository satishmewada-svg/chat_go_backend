package routes

import (
	"my-ecomm/controllers"
	"my-ecomm/middleware"
	"my-ecomm/services"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine) {
	authController := controllers.NewAuthController()
	productController := controllers.NewProductController()
	userController := controllers.NewUserController()
	chatController := controllers.NewChatController()
	presenceController := controllers.NewPresenceController() // NEW

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", func(c *gin.Context) {
				authController.Login(c)

				// Mark user as online after login
				if userID, exists := c.Get("userID"); exists {
					services.GetPresenceService().UserConnected(userID.(uint))
				}
			})
			auth.POST("/logout", middleware.AuthMiddleware(), func(c *gin.Context) {
				userID := c.GetUint("userID")

				// Mark user as offline
				services.GetPresenceService().UserDisconnected(userID)

				c.JSON(200, gin.H{"message": "Logged out successfully"})
			})
		}

		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			// Product routes
			protected.POST("/products", productController.CreateProduct)
			protected.GET("/products", productController.GetAllProductsUser)
			protected.GET("/products/:id", productController.GetProductById)

			// User routes
			protected.GET("/users", userController.GetAllUsers)
			protected.GET("/users/:id", userController.GetUserByID)

			// Presence routes (NEW)
			protected.POST("/presence/heartbeat", presenceController.Heartbeat)
			protected.POST("/presence/status", presenceController.GetOnlineStatus)
			protected.GET("/presence/online", presenceController.GetAllOnlineUsers)

			// Chat room routes
			protected.POST("/chat/rooms", chatController.CreateRoom)
			protected.POST("/chat/direct", chatController.CreateDirectChat)
			protected.GET("/chat/rooms", chatController.GetUserRooms)
			protected.GET("/chat/rooms/:id", chatController.GetRoomByID)
			protected.POST("/chat/rooms/:id/members", chatController.AddMemberToRoom)

			// Message routes
			protected.GET("/chat/rooms/:id/messages", chatController.GetRoomMessages)
			protected.POST("/chat/rooms/:id/messages", chatController.SendMessage)
			protected.PUT("/chat/messages/:id/read", chatController.MarkMessageAsRead)

			// WebSocket route
			protected.GET("/chat/rooms/:id/ws", chatController.HandleWebSocket)
		}
	}
}
