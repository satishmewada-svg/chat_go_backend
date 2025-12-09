package services

import (
	"errors"
	"my-ecomm/config"
	"my-ecomm/models"
)

type ProductService struct{}

func NewProductService() *ProductService {
	return &ProductService{}
}

func (s *ProductService) CreateProduct(name string, description string, price float64, stock int, UserID uint) (*models.Product, error) {
	product := models.Product{
		Name:        name,
		Description: description,
		Price:       price,
		Stock:       stock,
		UserID:      UserID,
	}
	if err := config.GetDB().Create(&product).Error; err != nil {
		return nil, errors.New("failed to create product")
	}
	return &product, nil
}

func (s *ProductService) GetProductByID(productID uint, userID uint) (*models.Product, error) {
	var product models.Product
	if err := config.GetDB().Where("id = ? AND user_id=?", productID, userID).Error; err != nil {
		return nil, errors.New("product not found")
	}
	return &product, nil
}
func (s *ProductService) GetAllProductsUser(userID uint) ([]models.Product, error) {
	var products []models.Product
	if err := config.GetDB().Where("user_id=?", userID).Find(&products).Error; err != nil {
		return nil, errors.New("failed to retrieve products")
	}
	return products, nil
}
