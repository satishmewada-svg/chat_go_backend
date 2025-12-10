// models/user.go
package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name       string         `gorm:"not null" json:"name"`
	Username   string         `gorm:"not null" json:"username"`
	Email      string         `gorm:"unique;not null" json:"email"`
	Password   string         `gorm:"not null" json:"-"`
	IsOnline   bool           `gorm:"default:false" json:"is_online"`
	LastSeenAt *time.Time     `json:"last_seen_at"`
	CreatedAt  time.Time      `json:"CreatedAt"`
	UpdatedAt  time.Time      `json:"UpdatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
