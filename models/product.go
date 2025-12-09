package models

import "gorm.io/gorm"

type Product struct {
	gorm.Model
	Name        string  `gorm:"not null" json:"name"`
	Description string  `json:"description"`
	Price       float64 `gorm:"not null" json:"price"`
	Stock       int     `gorm:"not null" json:"stock"`
	UserID      uint    `gorm:"not null" json:"user_id"`
	User        User    `json:"-" gorm:"foreignKey:UserID"`
}
