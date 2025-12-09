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
func (s *ChatService) CreateRoom(name, description string, createdBy uint, memberIDs []uint) (*models.ChatRoom, error) {
	room := models.ChatRoom{
		Name:        name,
		Description: description,
		CreatorID:   createdBy,
	}

	// Start transaction
	tx := config.GetDB().Begin()

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

	tx.Commit()

	// Reload room with members
	config.GetDB().Preload("Members").First(&room, room.ID)
	return &room, nil
}

// UpdateRoom updates a chat room's details
func (s *ChatService) UpdateRoom(roomID, userID uint, name, description *string) (*models.ChatRoom, error) {
	var room models.ChatRoom

	// First check if room exists and user has permission
	err := config.GetDB().First(&room, roomID).Error
	if err != nil {
		return nil, errors.New("room not found")
	}

	// Only group chats can be updated
	if !room.IsGroup {
		return nil, errors.New("cannot update direct chat")
	}

	// Verify user is a member of the room
	var count int64
	config.GetDB().
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

	if err := config.GetDB().Model(&room).Updates(updates).Error; err != nil {
		return nil, errors.New("failed to update room")
	}

	// Reload room with relationships
	config.GetDB().Preload("Members").Preload("Creator").First(&room, room.ID)

	return &room, nil
}

// UpdateMessage updates a message content
func (s *ChatService) UpdateMessage(messageID, userID uint, newContent string) (*models.Message, error) {
	var message models.Message

	// Find message and verify sender
	err := config.GetDB().First(&message, messageID).Error
	if err != nil {
		return nil, errors.New("message not found")
	}

	// Only the sender can update their message
	if message.SenderID != userID {
		return nil, errors.New("only sender can update the message")
	}

	// Update message content
	if err := config.GetDB().Model(&message).Update("content", newContent).Error; err != nil {
		return nil, errors.New("failed to update message")
	}

	// Reload message with sender info
	config.GetDB().Preload("Sender").First(&message, message.ID)

	return &message, nil
}

// DeleteMessage soft deletes a message
func (s *ChatService) DeleteMessage(messageID, userID uint) error {
	var message models.Message

	// Find message and verify sender
	err := config.GetDB().First(&message, messageID).Error
	if err != nil {
		return errors.New("message not found")
	}

	// Only the sender can delete their message
	if message.SenderID != userID {
		return errors.New("only sender can delete the message")
	}

	// Soft delete the message
	if err := config.GetDB().Delete(&message).Error; err != nil {
		return errors.New("failed to delete message")
	}

	return nil
}

// GetRoomByID gets a room by ID
func (s *ChatService) GetRoomByID(roomID uint, userID uint) (*models.ChatRoom, error) {
	var room models.ChatRoom

	// Check if user is a member of the room
	err := config.GetDB().
		Preload("Members").
		Joins("JOIN room_members ON room_members.chat_room_id = chat_rooms.id").
		Where("chat_rooms.id = ? AND room_members.user_id = ?", roomID, userID).
		First(&room).Error

	if err != nil {
		return nil, errors.New("room not found or access denied")
	}

	return &room, nil
}

// GetUserRooms gets all rooms for a user
func (s *ChatService) GetUserRooms(userID uint) ([]models.ChatRoom, error) {
	var rooms []models.ChatRoom

	err := config.GetDB().
		Preload("Members").
		Preload("Creator").
		Joins("JOIN room_members ON room_members.chat_room_id = chat_rooms.id").
		Where("room_members.user_id = ?", userID).
		Order("chat_rooms.updated_at DESC").
		Find(&rooms).Error

	if err != nil {
		return nil, errors.New("failed to retrieve rooms")
	}

	return rooms, nil
}

// SendMessage creates a new message
func (s *ChatService) SendMessage(content string, roomID, senderID uint) (*models.Message, error) {
	// Verify user is member of room
	var count int64
	config.GetDB().
		Table("room_members").
		Where("chat_room_id = ? AND user_id = ?", roomID, senderID).
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

	if err := config.GetDB().Create(&message).Error; err != nil {
		return nil, errors.New("failed to send message")
	}

	// Preload sender info
	config.GetDB().Preload("Sender").First(&message, message.ID)

	return &message, nil
}

// GetRoomMessages gets all messages for a room
func (s *ChatService) GetRoomMessages(roomID, userID uint, limit, offset int) ([]models.Message, error) {
	// Verify user is member of room
	var count int64
	config.GetDB().
		Table("room_members").
		Where("chat_room_id = ? AND user_id = ?", roomID, userID).
		Count(&count)

	if count == 0 {
		return nil, errors.New("access denied")
	}

	var messages []models.Message
	query := config.GetDB().
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

	// Verify user is member of the room
	err := config.GetDB().
		Joins("JOIN room_members ON room_members.chat_room_id = messages.room_id").
		Where("messages.id = ? AND room_members.user_id = ?", messageID, userID).
		First(&message).Error

	if err != nil {
		return errors.New("message not found or access denied")
	}

	return config.GetDB().Model(&message).Update("is_read", true).Error
}

// AddMemberToRoom adds a new member to a room
func (s *ChatService) AddMemberToRoom(roomID, userID, newMemberID uint) error {
	// Verify user is creator or member of room
	var room models.ChatRoom
	err := config.GetDB().
		Joins("JOIN room_members ON room_members.chat_room_id = chat_rooms.id").
		Where("chat_rooms.id = ? AND (chat_rooms.creator_id = ? OR room_members.user_id = ?)",
			roomID, userID, userID).
		First(&room).Error

	if err != nil {
		return errors.New("room not found or access denied")
	}

	// Add new member
	var newMember models.User
	if err := config.GetDB().First(&newMember, newMemberID).Error; err != nil {
		return errors.New("user not found")
	}

	if err := config.GetDB().Model(&room).Association("Members").Append(&newMember); err != nil {
		return errors.New("failed to add member")
	}

	return nil
}
