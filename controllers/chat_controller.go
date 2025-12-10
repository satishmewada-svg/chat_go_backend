// controllers/chat_controller.go
package controllers

import (
	"log"
	"my-ecomm/config"
	"my-ecomm/models"
	"my-ecomm/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ChatController struct {
	chatService *services.ChatService
	upgrader    websocket.Upgrader
}

func NewChatController() *ChatController {
	return &ChatController{
		chatService: services.NewChatService(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

type CreateRoomRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	MemberIDs   []uint `json:"member_ids"`
	IsGroup     *bool  `json:"is_group"` // Optional, defaults to true
}

type UpdateRoomRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type CreateDirectChatRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

// CreateRoom creates a new group chat room
func (cc *ChatController) CreateRoom(c *gin.Context) {
	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("userID")

	// Default to group chat if not specified
	isGroup := true
	if req.IsGroup != nil {
		isGroup = *req.IsGroup
	}

	// Create the room
	room := models.ChatRoom{
		Name:        req.Name,
		Description: req.Description,
		IsGroup:     isGroup,
		CreatorID:   userID,
	}

	if err := config.DB.Create(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create room"})
		return
	}

	// Add creator as member
	var creator models.User
	if err := config.DB.First(&creator, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := config.DB.Model(&room).Association("Members").Append(&creator); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add creator"})
		return
	}

	// Add additional members
	if len(req.MemberIDs) > 0 {
		var members []models.User
		if err := config.DB.Where("id IN ?", req.MemberIDs).Find(&members).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find members"})
			return
		}

		if err := config.DB.Model(&room).Association("Members").Append(&members); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add members"})
			return
		}
	}

	// Load the room with members
	config.DB.Preload("Members").Preload("Creator").First(&room, room.ID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Room created successfully",
		"room":    room,
	})
}

// UpdateRoom updates a chat room's name and/or description
func (cc *ChatController) UpdateRoom(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	var req UpdateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("userID")

	// Use service to update room
	room, err := cc.chatService.UpdateRoom(uint(roomID), userID, req.Name, req.Description)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "room not found" {
			status = http.StatusNotFound
		} else if err.Error() == "access denied" || err.Error() == "cannot update direct chat" {
			status = http.StatusForbidden
		} else if err.Error() == "no fields to update" {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Room updated successfully",
		"room":    room,
	})
}

// UpdateMessage updates a message's content
func (cc *ChatController) UpdateMessage(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var req UpdateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("userID")

	// Use service to update message
	message, err := cc.chatService.UpdateMessage(uint(messageID), userID, req.Content)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "message not found" {
			status = http.StatusNotFound
		} else if err.Error() == "only sender can update the message" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Message updated successfully",
		"data":    message,
	})
}

// DeleteMessage deletes a message
func (cc *ChatController) DeleteMessage(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	userID := c.GetUint("userID")

	// Use service to delete message
	err = cc.chatService.DeleteMessage(uint(messageID), userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "message not found" {
			status = http.StatusNotFound
		} else if err.Error() == "only sender can delete the message" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message deleted successfully"})
}

// GetUserRooms retrieves all rooms for the authenticated user
func (cc *ChatController) GetUserRooms(c *gin.Context) {
	userID := c.GetUint("userID")

	var roomIDs []uint
	err := config.DB.Table("room_members").
		Select("room_id").
		Where("user_id = ?", userID).
		Pluck("room_id", &roomIDs).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(roomIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"rooms": []models.ChatRoom{}})
		return
	}

	// Get rooms by IDs
	var rooms []models.ChatRoom
	err = config.DB.
		Where("id IN ?", roomIDs).
		Preload("Members").
		Preload("Creator").
		Order("updated_at DESC").
		Find(&rooms).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Load last message for each room
	for i := range rooms {
		var lastMessage models.Message
		if err := config.DB.Where("room_id = ?", rooms[i].ID).
			Order("created_at DESC").
			Limit(1).
			Preload("Sender").
			First(&lastMessage).Error; err == nil {
			rooms[i].Messages = []models.Message{lastMessage}
		}
	}

	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
}

// GetRoomByID retrieves a specific room by ID
func (cc *ChatController) GetRoomByID(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	userID := c.GetUint("userID")

	var room models.ChatRoom
	err = config.DB.
		Preload("Members").
		Preload("Creator").
		First(&room, roomID).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	// Check if user is a member
	var isMember bool
	for _, member := range room.Members {
		if member.ID == userID {
			isMember = true
			break
		}
	}

	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"room": room})
}

// AddMemberToRoom adds a user to a room
func (cc *ChatController) AddMemberToRoom(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUserID := c.GetUint("userID")

	var room models.ChatRoom
	if err := config.DB.Preload("Members").First(&room, roomID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	// Only allow adding members to group chats
	if !room.IsGroup {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot add members to direct chats"})
		return
	}

	// Check if current user is a member
	var isMember bool
	for _, member := range room.Members {
		if member.ID == currentUserID {
			isMember = true
			break
		}
	}

	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this room"})
		return
	}

	var newMember models.User
	if err := config.DB.First(&newMember, req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := config.DB.Model(&room).Association("Members").Append(&newMember); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member added successfully"})
}

// GetRoomMessages retrieves messages for a room
func (cc *ChatController) GetRoomMessages(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	userID := c.GetUint("userID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Check if user is a member
	var room models.ChatRoom
	if err := config.DB.Preload("Members").First(&room, roomID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	var isMember bool
	for _, member := range room.Members {
		if member.ID == userID {
			isMember = true
			break
		}
	}

	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this room"})
		return
	}

	var messages []models.Message
	err = config.DB.
		Where("room_id = ?", roomID).
		Preload("Sender").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// SendMessage sends a message to a room
func (cc *ChatController) SendMessage(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("userID")

	// Check if user is a member
	var room models.ChatRoom
	if err := config.DB.Preload("Members").First(&room, roomID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	var isMember bool
	for _, member := range room.Members {
		if member.ID == userID {
			isMember = true
			break
		}
	}

	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this room"})
		return
	}

	message := models.Message{
		RoomID:   uint(roomID),
		SenderID: userID,
		Content:  req.Content,
	}

	if err := config.DB.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	config.DB.Preload("Sender").First(&message, message.ID)

	// Update room's updated_at
	config.DB.Model(&room).Update("updated_at", message.CreatedAt)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Message sent successfully",
		"data":    message,
	})
}

// MarkMessageAsRead marks a message as read
func (cc *ChatController) MarkMessageAsRead(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var message models.Message
	if err := config.DB.First(&message, messageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	message.IsRead = true
	if err := config.DB.Save(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark message as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message marked as read"})
}

// HandleWebSocket handles WebSocket connections
func (cc *ChatController) HandleWebSocket(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	userID := c.GetUint("userID")

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	conn, err := cc.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	log.Printf("User joined: ID=%d, Name=%s, RoomID=%d\n", user.ID, user.Name, roomID)

	client := &services.Client{
		ID:       user.ID,
		Username: user.Name,
		RoomID:   uint(roomID),
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      services.GetHub(),
	}

	client.Hub.Register <- client

	go client.WritePump()
	client.ReadPump()
}

func (cc *ChatController) CreateDirectChat(c *gin.Context) {
	var req CreateDirectChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUserID := c.GetUint("userID")
	otherUserID := req.UserID

	// Prevent creating direct chat with self
	if currentUserID == otherUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot create direct chat with yourself"})
		return
	}

	// Check if direct chat already exists
	var existingRoom models.ChatRoom
	err := config.DB.
		Joins("JOIN room_members rm1 ON rm1.room_id = chat_rooms.id AND rm1.user_id = ?", currentUserID).
		Joins("JOIN room_members rm2 ON rm2.room_id = chat_rooms.id AND rm2.user_id = ?", otherUserID).
		Where("chat_rooms.is_group = ?", false).
		Preload("Members").
		Preload("Creator").
		First(&existingRoom).Error

	if err == nil {
		// Direct chat already exists
		c.JSON(http.StatusOK, gin.H{
			"message": "Direct chat already exists",
			"room":    existingRoom,
		})
		return
	}

	// Create new direct chat
	var otherUser models.User
	if err := config.DB.First(&otherUser, otherUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var currentUser models.User
	if err := config.DB.First(&currentUser, currentUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Current user not found"})
		return
	}

	// Create direct chat room (name not really used for direct chats)
	room := models.ChatRoom{
		Name:        currentUser.Name + " & " + otherUser.Name,
		Description: "",
		IsGroup:     false,
		CreatorID:   currentUserID,
	}

	if err := config.DB.Create(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create direct chat"})
		return
	}

	// Add both users as members
	if err := config.DB.Model(&room).Association("Members").Append([]models.User{currentUser, otherUser}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add members"})
		return
	}

	// Load the room with members
	config.DB.Preload("Members").Preload("Creator").First(&room, room.ID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Direct chat created successfully",
		"room":    room,
	})
}
