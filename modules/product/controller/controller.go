package controller

import (
	"go-api-starter/core/controller"
	authservice "go-api-starter/modules/auth/service"
	"go-api-starter/modules/product/service"
)

type ProductController struct {
	controller.BaseController
	ProductService service.ProductServiceInterface
	AuthService    authservice.AuthServiceInterface
}

func NewProductController(service service.ProductServiceInterface, authSvc authservice.AuthServiceInterface) *ProductController {
	return &ProductController{
		BaseController: controller.NewBaseController(),
		ProductService: service,
		AuthService:    authSvc,
	}
}
