// routes/routes.go
package routes

import (
	"my-ecomm/controllers"
	"my-ecomm/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine) {
	authController := controllers.NewAuthController()
	productController := &controllers.ProductController{}
	chatController := controllers.NewChatController()
	userController := controllers.NewUserController()

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", authController.Login)
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

			// Chat routes
			chat := protected.Group("/chat")
			{
				// Room routes
				chat.POST("/rooms", chatController.CreateRoom)
				chat.POST("/direct", chatController.CreateDirectChat)
				chat.GET("/rooms", chatController.GetUserRooms)
				chat.GET("/rooms/:id", chatController.GetRoomByID)
				chat.PUT("/rooms/:id", chatController.UpdateRoom)
				chat.POST("/rooms/:id/members", chatController.AddMemberToRoom)

				// Message routes
				chat.GET("/rooms/:id/messages", chatController.GetRoomMessages)
				chat.POST("/rooms/:id/messages", chatController.SendMessage)
				chat.PUT("/messages/:id", chatController.UpdateMessage)
				chat.DELETE("/messages/:id", chatController.DeleteMessage)
				chat.PUT("/messages/:id/read", chatController.MarkMessageAsRead)

				// WebSocket route
				chat.GET("/rooms/:id/ws", chatController.HandleWebSocket)
			}
		}
	}
}
