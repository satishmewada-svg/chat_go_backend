// models/chat.go
package models

import (
	"time"

	"gorm.io/gorm"
)

type ChatRoom struct {
	gorm.Model
	Name        string `json:"name" gorm:"not null"`
	Description string `json:"description"`
	IsGroup     bool   `json:"is_group" gorm:"default:true"`
	CreatorID   uint   `json:"creator_id" gorm:"not null"`
	Creator     User   `json:"creator" gorm:"foreignKey:CreatorID"`
	// CRITICAL FIX: Use joinForeignKey and Reference (not References)
	Members   []User         `json:"members" gorm:"many2many:room_members;foreignKey:ID;joinForeignKey:RoomID;References:ID;joinReferences:UserID"`
	Messages  []Message      `json:"messages,omitempty" gorm:"foreignKey:RoomID"`
	CreatedAt time.Time      `json:"CreatedAt"`
	UpdatedAt time.Time      `json:"UpdatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
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

// RoomMember - explicit join table
type RoomMember struct {
	RoomID    uint      `gorm:"primaryKey;column:room_id" json:"room_id"`
	UserID    uint      `gorm:"primaryKey;column:user_id" json:"user_id"`
	JoinedAt  time.Time `gorm:"autoCreateTime" json:"joined_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (RoomMember) TableName() string {
	return "room_members"
}
