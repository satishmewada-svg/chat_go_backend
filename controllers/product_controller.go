package controllers

import (
	"my-ecomm/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProductController struct {
	productService *services.ProductService
}

func NewProductController() *ProductController {
	return &ProductController{
		productService: services.NewProductService(),
	}
}

type CreateProductInput struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Stock       int     `json:"stock" binding:"required,gte=0"`
}

func (ctrl *ProductController) CreateProduct(c *gin.Context) {
	var input CreateProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	userID, _ := c.Get("userID")
	product, err := ctrl.productService.CreateProduct(input.Name, input.Description, input.Price, input.Stock, userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Product created successfully",
		"product": product,
	})
}

func (ctrl *ProductController) GetProductById(c *gin.Context) {
	productIDParam, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}
	userID, _ := c.Get("userID")
	product, err := ctrl.productService.GetProductByID(uint(productIDParam), userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"product": product,
	})
}
func (ctrl *ProductController) GetAllProductsUser(c *gin.Context) {
	userID, _ := c.Get("userID")
	products, err := ctrl.productService.GetAllProductsUser(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"products": products,
	})
}
