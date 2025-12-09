package services

import (
	"errors"
	"my-ecomm/config"
	"my-ecomm/models"
	"my-ecomm/utils"
)

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (s *AuthService) Register(email, password, name string) (*models.User, string, error) {
	var existingUser models.User
	if err := config.GetDB().Where("email = ?", email).First(&existingUser).Error; err == nil {
		return nil, "", errors.New("user already exists")
	}
	user := &models.User{
		Email:    email,
		Password: password,
		Name:     name,
	}
	if err := user.HashPassword(); err != nil {
		return nil, "", errors.New("failed to hash password")
	}
	if err := config.GetDB().Create(&user).Error; err != nil {
		return nil, "", errors.New("failed to create user")
	}
	token, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, "", errors.New("failed to generate token")
	}
	return user, token, nil
}

func (s *AuthService) Login(email, password string) (*models.User, string, error) {
	var user models.User
	if err := config.GetDB().Where("email= ?", email).First(&user).Error; err != nil {
		return nil, "", errors.New("invalid credentials")
	}
	if !user.CheckPassword(password) {
		return nil, "", errors.New("invalid password")
	}
	token, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, "", errors.New("failed to generate token")
	}
	return &user, token, nil
}
