// models/chat.go
package models

import (
	"time"

	"gorm.io/gorm"
)

type ChatRoom struct {
	gorm.Model
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	IsGroup     bool           `json:"is_group" gorm:"default:true"` // NEW: true for groups, false for direct
	CreatorID   uint           `json:"creator_id" gorm:"not null"`
	Creator     User           `json:"creator" gorm:"foreignKey:CreatorID"`
	Members     []User         `json:"members" gorm:"many2many:room_members;"`
	Messages    []Message      `json:"messages,omitempty" gorm:"foreignKey:RoomID"`
	CreatedAt   time.Time      `json:"CreatedAt"`
	UpdatedAt   time.Time      `json:"UpdatedAt"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

type Message struct {
	gorm.Model
	RoomID    uint           `json:"room_id" gorm:"not null;index"`
	Room      ChatRoom       `json:"-" gorm:"foreignKey:RoomID"`
	SenderID  uint           `json:"sender_id" gorm:"not null;index"`
	Sender    User           `json:"sender" gorm:"foreignKey:SenderID"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	IsRead    bool           `json:"is_read" gorm:"default:false"`
	CreatedAt time.Time      `json:"CreatedAt"`
	UpdatedAt time.Time      `json:"UpdatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type RoomMember struct {
	RoomID    uint      `json:"room_id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"primaryKey"`
	JoinedAt  time.Time `json:"joined_at" gorm:"autoCreateTime"`
	CreatedAt time.Time `json:"CreatedAt"`
	UpdatedAt time.Time `json:"UpdatedAt"`
}

func (RoomMember) TableName() string {
	return "room_members"
}
