// services/chat_service.go
package services

import (
	"errors"
	"my-ecomm/config"
	"my-ecomm/models"
)

type ChatService struct{}

func NewChatService() *ChatService {
	return &ChatService{}
}

// CreateRoom creates a new chat room
func (s *ChatService) CreateRoom(name, description string, createdBy uint, memberIDs []uint, isGroup bool) (*models.ChatRoom, error) {
	room := models.ChatRoom{
		Name:        name,
		Description: description,
		CreatorID:   createdBy,
		IsGroup:     isGroup,
	}

	// Start transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		return nil, errors.New("failed to start transaction")
	}

	if err := tx.Create(&room).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("failed to create chat room")
	}

	// Add creator as member
	memberIDs = append(memberIDs, createdBy)

	// Add members to room
	for _, memberID := range memberIDs {
		var user models.User
		if err := tx.First(&user, memberID).Error; err != nil {
			continue // Skip invalid user IDs
		}
		if err := tx.Model(&room).Association("Members").Append(&user); err != nil {
			tx.Rollback()
			return nil, errors.New("failed to add members to room")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, errors.New("failed to commit transaction")
	}

	// Reload room with members
	config.DB.Preload("Members").Preload("Creator").First(&room, room.ID)
	return &room, nil
}

// UpdateRoom updates a chat room's details
func (s *ChatService) UpdateRoom(roomID, userID uint, name, description *string) (*models.ChatRoom, error) {
	var room models.ChatRoom

	// First check if room exists
	if err := config.DB.First(&room, roomID).Error; err != nil {
		return nil, errors.New("room not found")
	}

	// Only group chats can be updated
	if !room.IsGroup {
		return nil, errors.New("cannot update direct chat")
	}

	// Verify user is a member of the room
	var count int64
	config.DB.
		Table("room_members").
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count)

	if count == 0 {
		return nil, errors.New("access denied")
	}

	// Update fields if provided
	updates := make(map[string]interface{})

	if name != nil && *name != "" {
		updates["name"] = *name
	}

	if description != nil {
		updates["description"] = *description
	}

	if len(updates) == 0 {
		return nil, errors.New("no fields to update")
	}

	if err := config.DB.Model(&room).Updates(updates).Error; err != nil {
		return nil, errors.New("failed to update room")
	}

	// Reload room with relationships
	config.DB.Preload("Members").Preload("Creator").First(&room, room.ID)

	return &room, nil
}

// UpdateMessage updates a message content
func (s *ChatService) UpdateMessage(messageID, userID uint, newContent string) (*models.Message, error) {
	var message models.Message

	// Find message and verify sender
	if err := config.DB.First(&message, messageID).Error; err != nil {
		return nil, errors.New("message not found")
	}

	// Only the sender can update their message
	if message.SenderID != userID {
		return nil, errors.New("only sender can update the message")
	}

	// Update message content
	if err := config.DB.Model(&message).Update("content", newContent).Error; err != nil {
		return nil, errors.New("failed to update message")
	}

	// Reload message with sender info
	config.DB.Preload("Sender").First(&message, message.ID)

	return &message, nil
}

// DeleteMessage soft deletes a message
func (s *ChatService) DeleteMessage(messageID, userID uint) error {
	var message models.Message

	// Find message and verify sender
	if err := config.DB.First(&message, messageID).Error; err != nil {
		return errors.New("message not found")
	}

	// Only the sender can delete their message
	if message.SenderID != userID {
		return errors.New("only sender can delete the message")
	}

	// Soft delete the message
	if err := config.DB.Delete(&message).Error; err != nil {
		return errors.New("failed to delete message")
	}

	return nil
}

// GetRoomByID gets a room by ID
func (s *ChatService) GetRoomByID(roomID uint, userID uint) (*models.ChatRoom, error) {
	var room models.ChatRoom

	// First get the room
	if err := config.DB.First(&room, roomID).Error; err != nil {
		return nil, errors.New("room not found")
	}

	// Check if user is a member
	var count int64
	config.DB.
		Table("room_members").
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count)

	if count == 0 {
		return nil, errors.New("access denied")
	}

	// Load relationships
	config.DB.Preload("Members").Preload("Creator").First(&room, room.ID)

	return &room, nil
}

// GetUserRooms gets all rooms for a user
func (s *ChatService) GetUserRooms(userID uint) ([]models.ChatRoom, error) {
	// Get room IDs where user is a member
	var roomIDs []uint
	if err := config.DB.
		Table("room_members").
		Where("user_id = ?", userID).
		Pluck("room_id", &roomIDs).Error; err != nil {
		return nil, errors.New("failed to retrieve rooms")
	}

	if len(roomIDs) == 0 {
		return []models.ChatRoom{}, nil
	}

	var rooms []models.ChatRoom
	if err := config.DB.
		Where("id IN ?", roomIDs).
		Preload("Members").
		Preload("Creator").
		Order("updated_at DESC").
		Find(&rooms).Error; err != nil {
		return nil, errors.New("failed to retrieve rooms")
	}

	return rooms, nil
}

// SendMessage creates a new message
func (s *ChatService) SendMessage(content string, roomID, senderID uint) (*models.Message, error) {
	// Verify user is member of room
	var count int64
	config.DB.
		Table("room_members").
		Where("room_id = ? AND user_id = ?", roomID, senderID).
		Count(&count)

	if count == 0 {
		return nil, errors.New("user is not a member of this room")
	}

	message := models.Message{
		Content:  content,
		RoomID:   roomID,
		SenderID: senderID,
		IsRead:   false,
	}

	if err := config.DB.Create(&message).Error; err != nil {
		return nil, errors.New("failed to send message")
	}

	// Preload sender info
	config.DB.Preload("Sender").First(&message, message.ID)

	return &message, nil
}

// GetRoomMessages gets all messages for a room
func (s *ChatService) GetRoomMessages(roomID, userID uint, limit, offset int) ([]models.Message, error) {
	// Verify user is member of room
	var count int64
	config.DB.
		Table("room_members").
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count)

	if count == 0 {
		return nil, errors.New("access denied")
	}

	var messages []models.Message
	query := config.DB.
		Preload("Sender").
		Where("room_id = ?", roomID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, errors.New("failed to retrieve messages")
	}

	return messages, nil
}

// MarkMessageAsRead marks a message as read
func (s *ChatService) MarkMessageAsRead(messageID, userID uint) error {
	var message models.Message

	// First get the message
	if err := config.DB.First(&message, messageID).Error; err != nil {
		return errors.New("message not found")
	}

	// Verify user is member of the room
	var count int64
	config.DB.
		Table("room_members").
		Where("room_id = ? AND user_id = ?", message.RoomID, userID).
		Count(&count)

	if count == 0 {
		return errors.New("access denied")
	}

	return config.DB.Model(&message).Update("is_read", true).Error
}

// AddMemberToRoom adds a new member to a room
func (s *ChatService) AddMemberToRoom(roomID, userID, newMemberID uint) error {
	var room models.ChatRoom

	// Get room and verify it exists
	if err := config.DB.First(&room, roomID).Error; err != nil {
		return errors.New("room not found")
	}

	// Verify user is member or creator of room
	var count int64
	config.DB.
		Table("room_members").
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count)

	isCreator := room.CreatorID == userID

	if count == 0 && !isCreator {
		return errors.New("access denied")
	}

	// Find the new member
	var newMember models.User
	if err := config.DB.First(&newMember, newMemberID).Error; err != nil {
		return errors.New("user not found")
	}

	// Check if already a member
	var existingCount int64
	config.DB.
		Table("room_members").
		Where("room_id = ? AND user_id = ?", roomID, newMemberID).
		Count(&existingCount)

	if existingCount > 0 {
		return errors.New("user is already a member")
	}

	// Add the member
	if err := config.DB.Model(&room).Association("Members").Append(&newMember); err != nil {
		return errors.New("failed to add member")
	}

	return nil
}
